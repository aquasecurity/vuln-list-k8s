/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package collector

import (
	"encoding/json"
	"fmt"
	"github.com/aquasecurity/vuln-list-k8s/collector/osv"
	"github.com/aquasecurity/vuln-list-k8s/collector/utils"
	"golang.org/x/xerrors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	k8svulnDBURL = "https://kubernetes.io/docs/reference/issues-security/official-cve-feed/index.json"
	mitreURL     = "https://cveawg.mitre.org/api/cve"
	cveList      = "https://www.cve.org/"
	vulnFolder   = "vulns"
)

type mitreOptions struct {
	mitreURL string
}

type mitreOption func(*mitreOptions)

func WithMitreURL(mitreURL string) mitreOption {
	return func(opts *mitreOptions) {
		opts.mitreURL = mitreURL
	}
}

type Updater struct {
	*mitreOptions
}

func NewUpdater(opts ...mitreOption) Updater {
	o := &mitreOptions{
		mitreURL: mitreURL,
	}
	for _, opt := range opts {
		opt(o)
	}
	return Updater{
		mitreOptions: o,
	}
}

type VulnDB struct {
	Cves []*osv.OSV
}

type CVE struct {
	Items []Item `json:"items,omitempty"`
}

type Item struct {
	ID            string `json:"id,omitempty"`
	Summary       string `json:"summary,omitempty"`
	ContentText   string `json:"content_text,omitempty"`
	DatePublished string `json:"date_published,omitempty"`
	ExternalURL   string `json:"external_url,omitempty"`
	URL           string `json:"url,omitempty"`
}

func (u Updater) Collect() (*VulnDB, error) {
	response, err := http.Get(k8svulnDBURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var db CVE
	if err = json.NewDecoder(response.Body).Decode(&db); err != nil {
		return nil, err
	}
	cvesMap, err := cveIDToModifiedMap(filepath.Join(utils.VulnListDir(), vulnFolder))
	if err != nil {
		return nil, err
	}
	return u.ParseVulnDBData(db, cvesMap)
}

const (
	// excludeNonCoreComponentsCves exclude  cves with missing data or non k8s core components
	excludeNonCoreComponentsCves = "CVE-2019-11255,CVE-2020-8554"
)

func (u Updater) Update() error {
	if err := u.update(); err != nil {
		return xerrors.Errorf("error in k8s update: %w", err)
	}
	return nil
}

func (u Updater) update() error {
	log.Printf("Fetching k8s cves")

	k8sdb, err := u.Collect()
	if err != nil {
		return err
	}
	for _, cve := range k8sdb.Cves {
		if err = utils.Write(filepath.Join(utils.VulnListDir(), vulnFolder, fmt.Sprintf("%s.json", cve.ID)), cve); err != nil {
			return xerrors.Errorf("failed to save k8s CVE detail: %w", err)
		}
	}

	return nil
}

func (u Updater) ParseVulnDBData(db CVE, cvesMap map[string]string) (*VulnDB, error) {
	var fullVulnerabilities []*osv.OSV
	for _, item := range db.Items {
		for _, cveID := range getMultiIDs(item.ID) {
			// check if the current cve is older than the existing one on the vuln-list-k8s repo
			if strings.Contains(excludeNonCoreComponentsCves, item.ID) || olderCve(cveID, item.DatePublished, cvesMap) {
				continue
			}
			vulnerability, err := parseMitreCve(item.ExternalURL, u.mitreURL, cveID)
			if err != nil {
				return nil, err
			}
			if cveMissingImportantData(vulnerability) {
				continue
			}
			descComponent := getComponentFromDescription(item.ContentText, vulnerability.Package)
			fullVulnerabilities = append(fullVulnerabilities, &osv.OSV{
				ID:        cveID,
				Modified:  item.DatePublished,
				Published: item.DatePublished,
				Summary:   item.Summary,
				Details:   vulnerability.Description,
				Affected:  getAffectedEvents(vulnerability.versions, getComponentName(descComponent, vulnerability.Package), vulnerability.CvssV3),
				References: []osv.Reference{
					{
						Url: item.URL, Type: "ADVISORY",
					}, {
						Url: item.ExternalURL, Type: "ADVISORY",
					},
				},
			})
		}
	}
	return &VulnDB{fullVulnerabilities}, nil
}

func getAffectedEvents(v []*Version, p string, cvss Cvssv3) []osv.Affected {
	events := make([]osv.Event, 0)
	for _, av := range v {
		if len(av.Introduced) == 0 {
			continue
		}
		if len(av.Introduced) > 0 {
			events = append(events, osv.Event{Introduced: av.Introduced})
		}
		if len(av.Fixed) > 0 {
			events = append(events, osv.Event{Fixed: av.Fixed})
		} else if len(av.LastAffected) > 0 {
			events = append(events, osv.Event{LastAffected: av.LastAffected})
		} else if len(av.Introduced) > 0 && len(av.LastAffected) == 0 && len(av.Fixed) == 0 {
			events = append(events, osv.Event{LastAffected: av.Introduced})
		}
	}
	return []osv.Affected{
		{
			Ranges: []osv.Range{
				{
					Events: events,
					Type:   "SEMVER",
				},
			},
			Package: osv.Package{
				Name:      p,
				Ecosystem: "kubernetes",
			},
			Severities: []osv.Severity{
				{
					Type:  cvss.Type,
					Score: cvss.Vector,
				},
			},
		},
	}

}

func getComponentName(k8sComponent string, mitreComponent string) string {
	if len(k8sComponent) == 0 {
		k8sComponent = mitreComponent
	}
	if strings.ToLower(mitreComponent) != "kubernetes" {
		k8sComponent = mitreComponent
	}
	return strings.ToLower(fmt.Sprintf("%s/%s", upstreamOrgByName(k8sComponent), upstreamRepoByName(k8sComponent)))
}

func cveMissingImportantData(vulnerability *Cve) bool {
	return len(vulnerability.versions) == 0 ||
		len(vulnerability.Package) == 0 ||
		len(vulnerability.CvssV3.Vector) == 0
}

// cveIDToModifiedMap read existing cves from vulnList folder and map it to cve id and last updated
func cveIDToModifiedMap(cveFolderPath string) (map[string]string, error) {
	mapCveTime := make(map[string]string)
	if _, err := os.Stat(cveFolderPath); os.IsNotExist(err) {
		return mapCveTime, nil
	}
	fileInfo, err := os.ReadDir(cveFolderPath)
	if err != nil {
		return mapCveTime, err
	}
	for _, file := range fileInfo {
		if file.IsDir() {
			continue
		}
		if !(strings.Contains(file.Name(), "CVE-") && strings.HasSuffix(file.Name(), ".json")) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(cveFolderPath, file.Name()))
		if err != nil {
			return nil, err
		}
		var cve osv.OSV
		err = json.Unmarshal([]byte(strings.ReplaceAll(string(b), "\n", "")), &cve)
		if err != nil {
			return nil, err
		}
		mapCveTime[cve.ID] = cve.Modified
	}
	return mapCveTime, nil
}

// olderCve check if the current cve is older than the existing one
func olderCve(cveID string, currentCVEUpdated string, existCveLastUpdated map[string]string) bool {
	if len(existCveLastUpdated) == 0 {
		return false
	}
	var lastUpdated string
	var ok bool
	if lastUpdated, ok = existCveLastUpdated[cveID]; !ok {
		return false
	}
	existLastUpdated, err := time.Parse(time.RFC3339, lastUpdated)
	if err != nil {
		return false
	}
	currentLastUpdated, err := time.Parse(time.RFC3339, currentCVEUpdated)
	if err != nil {
		return false
	}
	// check if the current collcted cve is older or same as the existing one
	if currentLastUpdated.Before(existLastUpdated) || currentLastUpdated == existLastUpdated {
		return true
	}

	return false
}

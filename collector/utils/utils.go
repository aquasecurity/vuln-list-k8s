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
package utils

import (
	"encoding/json"

	"os"
	"path/filepath"

	"golang.org/x/xerrors"
)

var vulnListDir = filepath.Join(CacheDir(), "vuln-list")

func CacheDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	dir := filepath.Join(cacheDir, "vuln-list-update")
	return dir
}

func SetVulnListDir(dir string) {
	vulnListDir = dir
}

func VulnListDir() string {
	return vulnListDir
}

func Write(filePath string, data interface{}) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return xerrors.Errorf("failed to create %s: %w", dir, err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return xerrors.Errorf("file create error: %w", err)
	}
	defer f.Close()

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return xerrors.Errorf("JSON marshal error: %w", err)
	}

	_, err = f.Write(b)
	if err != nil {
		return xerrors.Errorf("file write error: %w", err)
	}
	return nil
}

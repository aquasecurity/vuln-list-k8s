package main

import (
	"flag"
	"log"

	"github.com/aquasecurity/vuln-list-k8s/collector"
	"github.com/aquasecurity/vuln-list-k8s/collector/utils"
)

var (
	vulnListDir = flag.String("vuln-list-dir", "k8s", "vuln-list dir")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	flag.Parse()
	if *vulnListDir != "" {
		utils.SetVulnListDir(*vulnListDir)
	}
	return collector.NewUpdater().Update()
}

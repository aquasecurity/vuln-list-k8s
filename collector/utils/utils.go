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

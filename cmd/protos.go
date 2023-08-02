package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filename))
}

func findProtos() ([]string, error) {
	dir := filepath.Join(projectRoot(), "nexus")
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ".proto" {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

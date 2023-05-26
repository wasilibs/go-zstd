package main

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"os"
	"path/filepath"
)

// UpdateLibs updates the precompiled wasm libraries.
func UpdateLibs() error {
	if err := sh.RunV("docker", "build", "-t", "wasmbuild", "-f", filepath.Join("buildtools", "wasm", "Dockerfile"), "."); err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	return sh.RunV("docker", "run", "-it", "--rm", "-v", fmt.Sprintf("%s:/out", filepath.Join(wd, "internal", "wasm")), "wasmbuild")
}

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
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

func Format() error {
	if err := sh.RunV("go", "run", fmt.Sprintf("mvdan.cc/gofumpt@%s", verGoFumpt), "-l", "-w", "."); err != nil {
		return err
	}
	if err := sh.RunV("go", "run", fmt.Sprintf("github.com/rinchsan/gosimports/cmd/gosimports@%s", verGosImports), "-w",
		"-local", "github.com/wasilibs/go-zstd",
		"."); err != nil {
		return nil
	}
	return nil
}

func Lint() error {
	return sh.RunV("go", "run", fmt.Sprintf("github.com/golangci/golangci-lint/cmd/golangci-lint@%s", verGolangCILint), "run", "--timeout", "5m")
}

// Test runs unit tests
func Test() error {
	return sh.RunV("go", "test", "-v", "-timeout=20m", "./...")
}

// Check runs lint and tests.
func Check() {
	mg.SerialDeps(Lint, Test)
}

// Bench runs benchmarks.
func Bench() error {
	if _, err := os.Stat(filepath.Join("bench", "silesia", "dickens")); err != nil {
		fmt.Println("Downloading benchmark corpus, this may take some time")
		if err := sh.RunV("git", "submodule", "update", "--init"); err != nil {
			return err
		}
	}
	return sh.RunV("go", "test", "-bench=.", "-run=^$", "-timeout=60m", "./bench")
}

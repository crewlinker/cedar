// Package main provides repo automation using mage.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/samber/lo"
)

// init performs some sanity checks before running anything.
func init() {
	mustBeInRootIfNotTest()
}

// Dev groups commands for local development.
type Dev mg.Namespace

// Lint the codebase through static code analysis.
func (Dev) Lint() error {
	if err := sh.Run("golangci-lint", "run"); err != nil {
		return fmt.Errorf("failed to run golang-ci: %w", err)
	}

	return nil
}

// Test tests all the code using Gingo, with an empty label filter.
func (Dev) Test() error {
	return (Dev{}).TestSome("")
}

// Build the binary components.
func (Dev) Build() error {
	// On mac, the LLVM compiler we need to use needs to be installed with Homebrew (brew install llvm)
	// and specified in the environment variables.
	cargoEnv := map[string]string{}
	if runtime.GOOS == "darwin" {
		llvm := lo.Must(sh.Output("brew", "--prefix", "llvm"))
		cargoEnv["AR"] = filepath.Join(llvm, "bin", "llvm-ar")
		cargoEnv["CC"] = filepath.Join(llvm, "bin", "clang")
	}

	// then we can call cargo build, with our custom llvm path.
	if err := sh.RunWith(cargoEnv, "cargo", "build",
		"--manifest-path=cedarwasm/Cargo.toml",
		"--target=wasm32-unknown-unknown",
		"--release"); err != nil {
		return fmt.Errorf("failed to build rust wasm: %w", err)
	}

	// copy the result so we can embed it in the go File.
	if err := sh.Copy(
		filepath.Join("cedarwasm", "cedar.wasm"),
		filepath.Join("cedarwasm", "target", "wasm32-unknown-unknown", "release", "cedar.wasm"),
	); err != nil {
		return fmt.Errorf("failed to copy wasm: %w", err)
	}

	return nil
}

// BuildAndTest will build any binary components and run all the tests.
func (Dev) BuildAndTest() error {
	if err := (Dev{}).Build(); err != nil {
		return err
	}

	return (Dev{}).Test()
}

// TestSome tests the whole repo using Ginkgo test runner with label filters applied.
func (Dev) TestSome(labelFilter string) error {
	if err := sh.Run(
		"go", "run", "-mod=readonly", "github.com/onsi/ginkgo/v2/ginkgo",
		"-p", "-randomize-all", "--fail-on-pending", "--race", "--trace",
		"--junit-report=test-report.xml",
		"--label-filter", labelFilter,
		"./...",
	); err != nil {
		return fmt.Errorf("failed to run ginkgo: %w", err)
	}

	return nil
}

// mustBeInRootIfNotTest checks that the command is run in the project root.
func mustBeInRootIfNotTest() {
	if _, err := os.ReadFile("go.mod"); err != nil && !strings.Contains(strings.Join(os.Args, ""), "-test.") {
		panic("must be in project root, couldn't stat go.mod file: " + err.Error())
	}
}

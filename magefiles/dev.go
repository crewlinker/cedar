// Package main provides repo automation using mage.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
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

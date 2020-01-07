package tests

import (
	"flag"
	"testing"
)
import "fmt"
import "os"

//revive:disable:import-shadowing

// This function handled running of all the tests.
func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	minCoverageFlag := flag.Float64(
		"minimum-coverage",
		 // Default to 85% coverage requirement.
		0.85,
		"minimum coverage for passing tests from 0.0 (none) - 1.0 (all lines.)",
	)

	flag.Parse()

	// Run the tests
	testResults := m.Run()

	// testResults 0 means we've passed,
	// and CoverMode will be non empty if run with -cover
	if testResults == 0 && testing.CoverMode() != "" {
		coverageResult := testing.Coverage()
		if coverageResult < *minCoverageFlag {
			fmt.Println(
				"Tests passed but coverageResult of '", coverageResult, "' does " +
					"not meet minimum requirement of '", *minCoverageFlag, "'",
			)
			testResults = -1
		}
	}

	// Return with exit code.
	os.Exit(testResults)
}

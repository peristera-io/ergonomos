// Package features wires the Gherkin specs in this directory to godog so they
// run as acceptance tests (CLAUDE.md: specify → red → green → refactor). Each
// scenario drives a fresh ergonomos instance through its real HTTP surface.
package features

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format: "pretty",
			Paths:  []string{"."},
			Output: colors.Colored(os.Stdout),
			// Strict fails the run on undefined or pending steps, so a new
			// .feature with no step definitions goes red instead of passing
			// silently (the default treats undefined steps as non-fatal).
			Strict:   true,
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("there were failed, pending or undefined scenarios")
	}
}

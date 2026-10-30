package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestTopPodRejectsUnknownSort(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"top", "pod", "--sort-by", "name"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestTopPodRejectsNameWithSelectors(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	for _, args := range [][]string{
		{"top", "pod", "pod-a", "--selector", "app=test"},
		{"top", "pod", "pod-a", "--field-selector", "spec.nodeName=node-a"},
	} {
		request := cli.InspectionRequest{Args: args}
		clitest.Inspect(t.Context(), app, request).Require(t,
			clitest.InspectionRejected(),
			clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
		)
	}
}

func TestTopPodAcceptsAssignedShortSelector(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"top", "pod", "-l=app=test"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

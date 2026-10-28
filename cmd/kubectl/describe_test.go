package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestDescribeRejectsNamedResourceWithSelector(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{Args: []string{"describe", "pods", "pod-a", "--selector", "app=test"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestDescribeAllowsBulkResourceWithSelector(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{Args: []string{"describe", "pods", "--selector", "app=test"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

func TestDescribeRejectsResourceArgumentsWithFilename(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{Args: []string{"describe", "pods", "--filename", "pod.yaml"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

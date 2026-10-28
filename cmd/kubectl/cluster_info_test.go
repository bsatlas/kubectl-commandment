package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestClusterInfoDumpRejectsUnknownOutputFormat(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{Args: []string{"cluster-info", "dump", "--output", "bogus"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
	)
}

func TestClusterInfoDumpRequiresTemplateArgumentForTemplateOutput(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{Args: []string{"cluster-info", "dump", "--output", "jsonpath"}}

	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestClusterInfoDumpAcceptsTemplateWithDefaultOutput(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{Args: []string{"cluster-info", "dump", "--template", "{{.metadata.name}}"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

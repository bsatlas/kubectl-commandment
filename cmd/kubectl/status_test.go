package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRolloutStatusAcceptsRepeatedResources(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "status", "deployment/frontend", "statefulset/backend"},
	})
	proof.Require(t, clitest.InspectionReady())
}

func TestRolloutStatusRejectsNegativeRevision(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "status", "deployment/frontend", "--revision", "-1"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestApplySetLastAppliedRequiresFilename(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	readyRequest := cli.InspectionRequest{
		Args: []string{"apply", "set-last-applied", "-f", "object.yaml"},
	}
	clitest.Inspect(t.Context(), app, readyRequest).Require(t, clitest.InspectionReady())

	rejectedRequest := cli.InspectionRequest{
		Args: []string{"apply", "set-last-applied"},
	}
	clitest.Inspect(t.Context(), app, rejectedRequest).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
	)
}

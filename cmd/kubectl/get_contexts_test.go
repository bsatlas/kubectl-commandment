package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestConfigGetContextsOwnsItsOutputLanguage(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))

	readyRequest := cli.InspectionRequest{
		Args: []string{"config", "get-contexts", "missing", "also-missing", "--output=name"},
	}
	clitest.Inspect(t.Context(), app, readyRequest).Require(t, clitest.InspectionReady())

	rejectedRequest := cli.InspectionRequest{
		Args: []string{"config", "get-contexts", "--output=json"},
	}
	clitest.Inspect(t.Context(), app, rejectedRequest).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
	)
}

func TestConfigGetContextsDoesNotCompleteContextNames(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "get-contexts", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	}
	clitest.Complete(t.Context(), app, request).Require(t, clitest.CompletionItem("dev").Absent())
}

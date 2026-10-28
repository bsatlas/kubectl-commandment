package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestConfigRenameContextCompletesOnlyTheSourceName(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configRenameContextCommand(destinations.Kubeconfig))
	})

	inspectionRequest := cli.InspectionRequest{
		Args: []string{"--kubeconfig", path, "config", "rename-context", "missing", "new-name"},
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.InspectionReady())

	sourceCompletionRequest := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "rename-context", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	}
	clitest.Complete(t.Context(), app, sourceCompletionRequest).Require(t, clitest.CompletionItem("dev").Present())

	destinationCompletionRequest := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "rename-context", "dev", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 5,
			Byte:     1,
		},
	}
	clitest.Complete(t.Context(), app, destinationCompletionRequest).Require(t, clitest.CompletionItem("dev").Absent())
}

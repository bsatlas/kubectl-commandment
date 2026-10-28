package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCurrentContextAdmitsIgnoredArguments(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(currentContextCommand(destinations.Kubeconfig))
	})
	request := cli.InspectionRequest{
		Args: []string{"config", "current-context", "ignored", "arguments"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

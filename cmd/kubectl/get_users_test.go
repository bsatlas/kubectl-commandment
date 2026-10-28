package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestConfigGetUsersAdmitsIgnoredArguments(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configGetUsersCommand(destinations.Kubeconfig))
	})
	request := cli.InspectionRequest{
		Args: []string{"config", "get-users", "ignored", "arguments"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestConfigGetClustersAdmitsIgnoredArguments(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configGetClustersCommand(destinations.Kubeconfig))
	})
	request := cli.InspectionRequest{
		Args: []string{"config", "get-clusters", "ignored", "arguments"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

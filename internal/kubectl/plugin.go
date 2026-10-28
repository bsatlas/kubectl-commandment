package kubectl

import "github.com/abstractionsinc/commandment/cli"

var pluginCommand = cli.Command[struct{}]{
	Name: "plugin",
	Description: cli.Description{
		Short: "Provides utilities for interacting with plugins",
		Long:  "Provides utilities for interacting with plugins.\n\n Plugins provide extended functionality that is not part of the major command-line distribution. Please refer to the documentation and examples for more information about how write your own plugins.\n\n The easiest way to discover and install plugins is via the kubernetes sub-project krew: [krew.sigs.k8s.io]. To install krew, visit https://krew.sigs.k8s.io/docs/user-guide/setup/install",
	},
	Examples: []cli.Example{
		{
			Description: "List all available plugins",
			CommandLine: "kubectl plugin list",
		},
		{
			Description: "List only binary names of available plugins without paths",
			CommandLine: "kubectl plugin list --name-only",
		},
	},
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cb.Subcommands.Command(pluginListCommand)
	},
}

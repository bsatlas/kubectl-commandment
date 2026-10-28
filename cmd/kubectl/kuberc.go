package main

import "github.com/abstractionsinc/commandment/cli"

var kubercCommand = cli.Command[struct{}]{
	Name: "kuberc",
	Description: cli.Description{
		Short: "Manage kuberc configuration files",
		Long:  "Manage user preferences (kuberc) file.\n\n The kuberc file allows you to customize your kubectl experience.",
	},
	Examples: []cli.Example{
		{
			Description: "View the current kuberc configuration",
			CommandLine: "kubectl kuberc view",
		},
		{
			Description: "Set a default value for a command flag",
			CommandLine: "kubectl kuberc set --section defaults --command get --option output=wide",
		},
		{
			Description: "Create an alias for a command",
			CommandLine: "kubectl kuberc set --section aliases --name getn --command get --prependarg nodes --option output=wide",
		},
	},
	Declare: func(cb *cli.CommandBuilder[struct{}]) {
		cb.Subcommands.Command(kubercSetCommand)
		cb.Subcommands.Command(kubercViewCommand)
	},
}

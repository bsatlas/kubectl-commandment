package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type pluginListConfig struct {
	Streams  genericiooptions.IOStreams
	NameOnly bool
}

var pluginListCommand = cli.Command[pluginListConfig]{
	Name: "list",
	Description: cli.Description{
		Short: "List all visible plugin executables on a user's PATH",
		Long:  "List all available plugin files on a user's PATH. To see plugins binary names without the full path use --name-only flag.\n\n Available plugin files are those that are: - executable - anywhere on the user's PATH - begin with \"kubectl-\"",
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
	Declare: func(cb *cli.CommandBuilder[pluginListConfig]) {
		cb.Access.Stdout(&cb.Config.Streams.Out)
		cb.Access.Stderr(&cb.Config.Streams.ErrOut)
		nameOnlyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NameOnly))
		nameOnlyFlag = nameOnlyFlag.Description("If true, display only the binary name of each plugin, rather than its full path")
		nameOnlyFlag.Long("name-only").ImplicitValue(true)
	},
	Run: runPluginList,
}

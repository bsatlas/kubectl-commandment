package kubectl

import "github.com/abstractionsinc/commandment/cli"

var optionsCommand = cli.Command[struct{}]{
	Name: "options",
	Description: cli.Description{
		Short: "Print the list of flags inherited by all commands",
		Long:  "Print the list of flags inherited by all commands",
	},
	Examples: []cli.Example{
		{
			Description: "Print flags inherited by all commands",
			CommandLine: "kubectl options",
		},
	},
}

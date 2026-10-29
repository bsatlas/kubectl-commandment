package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type currentContextConfig struct {
	Ignored    []string
	Kubeconfig string
	Stdout     io.Writer
}

func currentContextCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[currentContextConfig] {
	cmd := cli.Command[currentContextConfig]{
		Name: "current-context",
		Description: cli.Description{
			Short: "Display the current-context",
			Long:  "Display the current-context.",
		},
		Examples: []cli.Example{
			{
				Description: "Display the current-context",
				CommandLine: "kubectl config current-context",
			},
		},
		Declare: func(command *cli.CommandBuilder[currentContextConfig]) {
			command.Access.Value(kubeconfig, &command.Config.Kubeconfig)
			command.Access.Stdout(&command.Config.Stdout)
			ignored := command.Values.Sequence(&command.Config.Ignored, cli.Value.String())
			command.Arguments.Remainder(ignored, "ARG", cli.Count.ZeroOrMore()).Hidden()
		},
		Run: runCurrentContext,
	}
	return cmd
}

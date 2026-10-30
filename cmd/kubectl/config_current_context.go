package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type configCurrentContextConfig struct {
	Ignored    []string
	Kubeconfig string
	Stdout     io.Writer
}

func configCurrentContextCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configCurrentContextConfig] {
	cmd := cli.Command[configCurrentContextConfig]{
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
		Declare: func(command *cli.CommandBuilder[configCurrentContextConfig]) {
			command.Access.Value(kubeconfig, &command.Config.Kubeconfig)
			command.Access.Stdout(&command.Config.Stdout)
			ignored := command.Values.Sequence(&command.Config.Ignored, cli.Value.String())
			command.Arguments.Remainder(ignored, "ARG", cli.Count.ZeroOrMore()).Hidden()
		},
		Run: func(_ context.Context, cfg *configCurrentContextConfig) error {
			starting, err := configAccess(cfg.Kubeconfig).GetStartingConfig()
			if err != nil {
				return err
			}
			if starting.CurrentContext == "" {
				return fmt.Errorf("current-context is not set")
			}
			_, err = fmt.Fprintln(cfg.Stdout, starting.CurrentContext)
			return err
		},
	}
	return cmd
}

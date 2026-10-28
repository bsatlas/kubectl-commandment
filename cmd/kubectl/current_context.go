package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type currentContextConfig struct {
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
		},
		Run: runCurrentContext,
	}
	return cmd
}

func runCurrentContext(_ context.Context, cfg *currentContextConfig) error {
	starting, err := configAccess(cfg.Kubeconfig).GetStartingConfig()
	if err != nil {
		return err
	}
	if starting.CurrentContext == "" {
		return fmt.Errorf("current-context is not set")
	}
	_, err = fmt.Fprintln(cfg.Stdout, starting.CurrentContext)
	return err
}

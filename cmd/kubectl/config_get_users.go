package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type configGetUsersConfig struct {
	Ignored    []string
	Kubeconfig string
	Stdout     io.Writer
}

func configGetUsersCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configGetUsersConfig] {
	cmd := cli.Command[configGetUsersConfig]{
		Name: "get-users",
		Description: cli.Description{
			Short: "Display users defined in the kubeconfig",
			Long:  "Display users defined in the kubeconfig.",
		},
		Examples: []cli.Example{
			{
				Description: "List the users that kubectl knows about",
				CommandLine: "kubectl config get-users",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configGetUsersConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)
			ignored := cb.Values.Sequence(&cb.Config.Ignored, cli.Value.String())
			cb.Arguments.Remainder(ignored, "ARG", cli.Count.ZeroOrMore()).Hidden()
		},
		Run: func(_ context.Context, cfg *configGetUsersConfig) error {
			names, err := userNames(cfg.Kubeconfig)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cfg.Stdout, "NAME"); err != nil {
				return err
			}
			for _, name := range names {
				if _, err := fmt.Fprintln(cfg.Stdout, name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

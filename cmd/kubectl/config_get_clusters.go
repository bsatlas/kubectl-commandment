package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type configGetClustersConfig struct {
	Ignored    []string
	Kubeconfig string
	Stdout     io.Writer
}

func configGetClustersCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configGetClustersConfig] {
	cmd := cli.Command[configGetClustersConfig]{
		Name: "get-clusters",
		Description: cli.Description{
			Short: "Display clusters defined in the kubeconfig",
			Long:  "Display clusters defined in the kubeconfig.",
		},
		Examples: []cli.Example{
			{
				Description: "List the clusters that kubectl knows about",
				CommandLine: "kubectl config get-clusters",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configGetClustersConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)
			ignored := cb.Values.Sequence(&cb.Config.Ignored, cli.Value.String())
			cb.Arguments.Remainder(ignored, "ARG", cli.Count.ZeroOrMore()).Hidden()
		},
		Run: func(_ context.Context, cfg *configGetClustersConfig) error {
			config, err := configAccess(cfg.Kubeconfig).GetStartingConfig()
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cfg.Stdout, "NAME"); err != nil {
				return err
			}
			for name := range config.Clusters {
				if _, err := fmt.Fprintln(cfg.Stdout, name); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}

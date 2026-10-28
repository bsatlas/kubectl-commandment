package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type configGetClustersConfig struct {
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
		},
		Run: runConfigGetClusters,
	}
	return cmd
}

func runConfigGetClusters(_ context.Context, cfg *configGetClustersConfig) error {
	names, err := clusterNames(cfg.Kubeconfig)
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
}

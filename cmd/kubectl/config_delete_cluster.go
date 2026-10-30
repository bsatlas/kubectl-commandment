package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configDeleteClusterConfig struct {
	Kubeconfig string
	Name       string
	Stdout     io.Writer
}

func configDeleteClusterCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configDeleteClusterConfig] {
	cmd := cli.Command[configDeleteClusterConfig]{
		Name: "delete-cluster",
		Description: cli.Description{
			Short: "Delete the specified cluster from the kubeconfig",
			Long:  "Delete the specified cluster from the kubeconfig.",
		},
		Examples: []cli.Example{
			{
				Description: "Delete the minikube cluster",
				CommandLine: "kubectl config delete-cluster minikube",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configDeleteClusterConfig]) {
			kubeconfig := deriveKubeconfig(cb, &cb.Config.Kubeconfig, kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.ValueFor(name, "NAME").
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *configDeleteClusterConfig) string { return cfg.Kubeconfig },
					kubeconfigClusterName,
				)).
				From(kubeconfig)
		},
		Run: func(_ context.Context, cfg *configDeleteClusterConfig) error {
			access := configAccess(cfg.Kubeconfig)
			config, err := access.GetStartingConfig()
			if err != nil {
				return err
			}
			file := configFile(access)
			if _, ok := config.Clusters[cfg.Name]; !ok {
				return fmt.Errorf("cannot delete cluster %s, not in %s", cfg.Name, file)
			}
			delete(config.Clusters, cfg.Name)
			if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cfg.Stdout, "deleted cluster %s from %s\n", cfg.Name, file)
			return err
		},
	}
	return cmd
}

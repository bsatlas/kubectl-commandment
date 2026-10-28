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
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			clusters := cb.Values.Domain(func(_ context.Context, _ *configDeleteClusterConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
				path, ok := dependencies.Get(kubeconfig)
				if !ok {
					return fmt.Errorf("kubeconfig dependency is unavailable")
				}
				names, err := clusterNames(path)
				if err != nil {
					return err
				}
				for _, name := range names {
					domain.Value(name, name)
				}
				return nil
			}, kubeconfig)
			name := cb.Values.Value(&cb.Config.Name, clusters)
			cb.Arguments.ValueFor(name, "NAME")
		},
		Run: runConfigDeleteCluster,
	}
	return cmd
}

func runConfigDeleteCluster(_ context.Context, cfg *configDeleteClusterConfig) error {
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
}

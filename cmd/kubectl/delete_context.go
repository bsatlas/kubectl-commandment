package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configDeleteContextConfig struct {
	Kubeconfig string
	Name       string
	Stdout     io.Writer
	Stderr     io.Writer
}

func configDeleteContextCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configDeleteContextConfig] {
	cmd := cli.Command[configDeleteContextConfig]{
		Name: "delete-context",
		Description: cli.Description{
			Short: "Delete the specified context from the kubeconfig",
			Long:  "Delete the specified context from the kubeconfig.",
		},
		Examples: []cli.Example{
			{
				Description: "Delete the context for the minikube cluster",
				CommandLine: "kubectl config delete-context minikube",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configDeleteContextConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			contexts := cb.Values.Domain(func(_ context.Context, _ *configDeleteContextConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
				path, ok := dependencies.Get(kubeconfig)
				if !ok {
					return fmt.Errorf("kubeconfig dependency is unavailable")
				}
				names, err := contextNames(path)
				if err != nil {
					return err
				}
				for _, name := range names {
					domain.Value(name, name)
				}
				return nil
			}, kubeconfig)
			name := cb.Values.Value(&cb.Config.Name, contexts)
			cb.Arguments.ValueFor(name, "NAME")
		},
		Run: runConfigDeleteContext,
	}
	return cmd
}

func runConfigDeleteContext(_ context.Context, cfg *configDeleteContextConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	file := configFile(access)
	if _, ok := config.Contexts[cfg.Name]; !ok {
		return fmt.Errorf("cannot delete context %s, not in %s", cfg.Name, file)
	}
	if config.CurrentContext == cfg.Name {
		if _, err := fmt.Fprint(cfg.Stderr, "warning: this removed your active context, use \"kubectl config use-context\" to select a different one\n"); err != nil {
			return err
		}
	}
	delete(config.Contexts, cfg.Name)
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "deleted context %s from %s\n", cfg.Name, file)
	return err
}

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
			kubeconfig := deriveKubeconfig(cb, &cb.Config.Kubeconfig, kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.ValueFor(name, "NAME").
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *configDeleteContextConfig) string { return cfg.Kubeconfig },
					kubeconfigContextName,
				)).
				From(kubeconfig)
		},
		Run: func(_ context.Context, cfg *configDeleteContextConfig) error {
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
		},
	}
	return cmd
}

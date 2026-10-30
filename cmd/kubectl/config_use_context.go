package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configUseContextConfig struct {
	Kubeconfig string
	Context    string
	Stdout     io.Writer
}

func configUseContextCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configUseContextConfig] {
	cmd := cli.Command[configUseContextConfig]{
		Name:    "use-context",
		Aliases: cli.Aliases("use"),
		Description: cli.Description{
			Short: "Set the current-context in a kubeconfig file",
			Long:  "Set the current-context in a kubeconfig file.",
		},
		Examples: []cli.Example{
			{
				Description: "Use the context for the minikube cluster",
				CommandLine: "kubectl config use-context minikube",
			},
		},
		Declare: func(command *cli.CommandBuilder[configUseContextConfig]) {
			kubeconfig := deriveKubeconfig(command, &command.Config.Kubeconfig, kubeconfig)
			command.Access.Stdout(&command.Config.Stdout)

			contextName := command.Values.String(&command.Config.Context)
			command.Arguments.ValueFor(contextName, "CONTEXT_NAME").
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *configUseContextConfig) string { return cfg.Kubeconfig },
					kubeconfigContextName,
				)).
				From(kubeconfig)
			command.Rule.NotEmpty(contextName)
		},
		Run: func(_ context.Context, cfg *configUseContextConfig) error {
			access := configAccess(cfg.Kubeconfig)
			starting, err := access.GetStartingConfig()
			if err != nil {
				return err
			}
			if _, exists := starting.Contexts[cfg.Context]; !exists {
				return fmt.Errorf("no context exists with the name: %q", cfg.Context)
			}
			starting.CurrentContext = cfg.Context
			if err := clientcmd.ModifyConfig(access, *starting, true); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cfg.Stdout, "Switched to context %q.\n", cfg.Context)
			return err
		},
	}
	return cmd
}

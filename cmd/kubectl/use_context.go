package main

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type useContextConfig struct {
	Kubeconfig string
	Context    string
	Stdout     io.Writer
}

func useContextCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[useContextConfig] {
	cmd := cli.Command[useContextConfig]{
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
		Declare: func(command *cli.CommandBuilder[useContextConfig]) {
			command.Access.Value(kubeconfig, &command.Config.Kubeconfig)
			command.Access.Stdout(&command.Config.Stdout)

			contexts := command.Values.Domain(
				func(_ context.Context, _ *useContextConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
					path, ok := dependencies.Get(kubeconfig)
					if !ok {
						return fmt.Errorf("kubeconfig dependency is unavailable")
					}
					starting, err := configAccess(path).GetStartingConfig()
					if err != nil {
						return err
					}
					names := make([]string, 0, len(starting.Contexts))
					for name := range starting.Contexts {
						names = append(names, name)
					}
					slices.Sort(names)
					for _, name := range names {
						domain.Value(name, name)
					}
					return nil
				},
				kubeconfig,
			)
			contextName := command.Values.Value(&command.Config.Context, contexts)
			command.Arguments.ValueFor(contextName, "CONTEXT_NAME")
		},
		Run: runUseContext,
	}
	return cmd
}

func runUseContext(_ context.Context, cfg *useContextConfig) error {
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
}

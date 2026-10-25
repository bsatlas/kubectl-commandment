package config

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/kubectl-commandment/internal/root"
	"k8s.io/client-go/tools/clientcmd"
)

type useContextConfig struct {
	Kubeconfig string
	Context    string
	Stdout     io.Writer
}

type currentContextConfig struct {
	Kubeconfig string
	Stdout     io.Writer
}

// Command returns kubectl's static config command namespace with the services it needs bound as
// ordinary Go dependencies. kubeconfig is the root semantic destination consumed by descendants.
func Command(kubeconfig cli.Destination[root.Config, string], services Services) cli.Command[struct{}] {
	return cli.Command[struct{}]{
		Name:        "config",
		Description: cli.Description{Short: "Modify kubeconfig files."},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			useContext := useContextCommand(kubeconfig, services)
			cb.Subcommands.Command(useContext)

			currentContext := currentContextCommand(kubeconfig, services)
			cb.Subcommands.Command(currentContext)
		},
	}
}

func useContextCommand(kubeconfig cli.Destination[root.Config, string], services Services) cli.Command[useContextConfig] {
	return cli.Command[useContextConfig]{
		Name:        "use-context",
		Aliases:     cli.Aliases("use"),
		Description: cli.Description{Short: "Set the current-context in a kubeconfig file."},
		Declare: func(cb *cli.CommandBuilder[useContextConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			contexts := cb.Values.Domain(
				func(_ context.Context, _ *useContextConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
					path, ok := dependencies.Get(kubeconfig)
					if !ok {
						return fmt.Errorf("kubeconfig dependency is unavailable")
					}
					starting, err := services.configAccess(path).GetStartingConfig()
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
			contextName := cb.Values.Value(&cb.Config.Context, contexts)
			cb.Arguments.ValueFor(contextName, "CONTEXT_NAME")
		},
		Run: func(_ context.Context, cfg *useContextConfig) error {
			access := services.configAccess(cfg.Kubeconfig)
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
}

func currentContextCommand(kubeconfig cli.Destination[root.Config, string], services Services) cli.Command[currentContextConfig] {
	return cli.Command[currentContextConfig]{
		Name:        "current-context",
		Description: cli.Description{Short: "Display the current-context"},
		Declare: func(cb *cli.CommandBuilder[currentContextConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)
		},
		Run: func(_ context.Context, cfg *currentContextConfig) error {
			starting, err := services.configAccess(cfg.Kubeconfig).GetStartingConfig()
			if err != nil {
				return err
			}
			if starting.CurrentContext == "" {
				return fmt.Errorf("current-context is not set")
			}
			_, err = fmt.Fprintln(cfg.Stdout, starting.CurrentContext)
			return err
		},
	}
}

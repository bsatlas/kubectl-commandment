package main

import (
	"context"
	"fmt"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
)

type configDeleteUserConfig struct {
	Kubeconfig string
	Name       string
	Stdout     io.Writer
}

func configDeleteUserCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configDeleteUserConfig] {
	cmd := cli.Command[configDeleteUserConfig]{
		Name: "delete-user",
		Description: cli.Description{
			Short: "Delete the specified user from the kubeconfig",
			Long:  "Delete the specified user from the kubeconfig.",
		},
		Examples: []cli.Example{
			{
				Description: "Delete the minikube user",
				CommandLine: "kubectl config delete-user minikube",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configDeleteUserConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			users := cb.Values.Domain(func(_ context.Context, _ *configDeleteUserConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
				path, ok := dependencies.Get(kubeconfig)
				if !ok {
					return fmt.Errorf("kubeconfig dependency is unavailable")
				}
				names, err := userNames(path)
				if err != nil {
					return err
				}
				for _, name := range names {
					domain.Value(name, name)
				}
				return nil
			}, kubeconfig)
			name := cb.Values.Value(&cb.Config.Name, users)
			cb.Arguments.ValueFor(name, "NAME")
		},
		Run: runConfigDeleteUser,
	}
	return cmd
}

func runConfigDeleteUser(_ context.Context, cfg *configDeleteUserConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	file := configFile(access)
	if _, ok := config.AuthInfos[cfg.Name]; !ok {
		return fmt.Errorf("cannot delete user %s, not in %s", cfg.Name, file)
	}
	delete(config.AuthInfos, cfg.Name)
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "deleted user %s from %s\n", cfg.Name, file)
	return err
}

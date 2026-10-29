package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
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
			kubeconfig := deriveKubeconfig(cb, &cb.Config.Kubeconfig, kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.ValueFor(name, "NAME").
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *configDeleteUserConfig) string { return cfg.Kubeconfig },
					kubeconfigUserName,
				)).
				From(kubeconfig)
		},
		Run: runConfigDeleteUser,
	}
	return cmd
}

package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
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
			kubeconfig := deriveKubeconfig(command, &command.Config.Kubeconfig, kubeconfig)
			command.Access.Stdout(&command.Config.Stdout)

			contextName := command.Values.String(&command.Config.Context)
			command.Arguments.ValueFor(contextName, "CONTEXT_NAME").
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *useContextConfig) string { return cfg.Kubeconfig },
					kubeconfigContextName,
				)).
				From(kubeconfig)
			command.Rule.NotEmpty(contextName)
		},
		Run: runUseContext,
	}
	return cmd
}

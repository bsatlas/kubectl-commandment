package kubectl

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
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
		Run: runConfigDeleteContext,
	}
	return cmd
}

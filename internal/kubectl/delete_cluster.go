package kubectl

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
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
			kubeconfig := deriveKubeconfig(cb, &cb.Config.Kubeconfig, kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.ValueFor(name, "NAME").
				CompleteBy(kubeconfigNameCompleter(
					func(cfg *configDeleteClusterConfig) string { return cfg.Kubeconfig },
					kubeconfigClusterName,
				)).
				From(kubeconfig)
		},
		Run: runConfigDeleteCluster,
	}
	return cmd
}

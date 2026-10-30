package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/clusterinfo"
)

type clusterInfoConfig struct {
	Client      clientConfig
	Streams     genericiooptions.IOStreams
	IgnoredArgs []string
}

func clusterInfoCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[clusterInfoConfig] {
	cmd := cli.Command[clusterInfoConfig]{
		Name: "cluster-info",
		Description: cli.Description{
			Short: "Display cluster information",
			Long:  "Display addresses of the control plane and services with label kubernetes.io/cluster-service=true. To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.",
		},
		Examples: []cli.Example{
			{
				Description: "Print the address of the control plane and cluster services",
				CommandLine: "kubectl cluster-info",
			},
		},
		Declare: func(cb *cli.CommandBuilder[clusterInfoConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			ignoredArgs := cb.Values.Sequence(&cb.Config.IgnoredArgs, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[clusterInfoConfig]) {
				form.Sequence(ignoredArgs, "ARG", cli.Count.ZeroOrMore()).Hidden()
			})

			dump := clusterInfoDumpCommand(client)
			cb.Subcommands.Command(dump)
		},
		Run: func(_ context.Context, cfg *clusterInfoConfig) error {
			getter := restClientGetter(cfg.Client)
			restConfig, err := getter.ToRESTConfig()
			if err != nil {
				return err
			}
			namespace := cfg.Client.Namespace
			if namespace == "" {
				namespace = metav1.NamespaceSystem
			}
			options := clusterinfo.ClusterInfoOptions{
				IOStreams: cfg.Streams,
				Namespace: namespace,
				Builder:   resource.NewBuilder(getter),
				Client:    restConfig,
			}
			return options.Run()
		},
	}
	return cmd
}

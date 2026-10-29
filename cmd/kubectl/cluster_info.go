package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
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
		Run: runClusterInfo,
	}
	return cmd
}

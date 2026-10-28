package main

import "github.com/abstractionsinc/commandment/cli"

func createServiceCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name:    "service",
		Aliases: cli.Aliases("svc"),
		Description: cli.Description{
			Short: "Create a service using a specified subcommand",
			Long:  "Create a service using a specified subcommand.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			clusterIP := createServiceClusterIPCommand(client)
			cb.Subcommands.Command(clusterIP)
			externalName := createServiceExternalNameCommand(client)
			cb.Subcommands.Command(externalName)
			loadBalancer := createServiceLoadBalancerCommand(client)
			cb.Subcommands.Command(loadBalancer)
			nodePort := createServiceNodePortCommand(client)
			cb.Subcommands.Command(nodePort)
		},
	}
	return cmd
}

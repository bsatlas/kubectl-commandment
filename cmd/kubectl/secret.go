package main

import (
	"github.com/abstractionsinc/commandment/cli"
)

func createSecretCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "secret",
		Description: cli.Description{
			Short: "Create a secret using a specified subcommand",
			Long:  "Create a secret with specified type.\n\n A docker-registry type secret is for accessing a container registry.\n\n A generic type secret indicate an Opaque secret type.\n\n A tls type secret holds TLS certificate and its associated key.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			dockerRegistry := createSecretDockerRegistryCommand(client)
			cb.Subcommands.Command(dockerRegistry)
			generic := createSecretGenericCommand(client)
			cb.Subcommands.Command(generic)
			tls := createSecretTLSCommand(client)
			cb.Subcommands.Command(tls)
		},
	}
	return cmd
}

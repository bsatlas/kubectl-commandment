package main

import (
	"github.com/abstractionsinc/commandment/cli"
)

func authCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "auth",
		Description: cli.Description{
			Short: "Inspect authorization",
			Long:  "Inspect authorization.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			canI := authCanICommand(client)
			reconcile := authReconcileCommand(client)
			whoAmI := authWhoAmICommand(client)
			cb.Subcommands.Command(canI)
			cb.Subcommands.Command(reconcile)
			cb.Subcommands.Command(whoAmI)
		},
	}
	return cmd
}

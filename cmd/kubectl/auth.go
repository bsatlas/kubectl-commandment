package main

import "github.com/abstractionsinc/commandment/cli"

func authCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "auth",
		Description: cli.Description{
			Short: "Inspect authorization",
			Long:  "Inspect authorization.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			canI := authCanICommand(client)
			cb.Subcommands.Command(canI)
			reconcile := authReconcileCommand(client)
			cb.Subcommands.Command(reconcile)

			whoami := authWhoamiCommand(client)
			cb.Subcommands.Command(whoami)
		},
	}
	return cmd
}

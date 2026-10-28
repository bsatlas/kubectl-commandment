package main

import "github.com/abstractionsinc/commandment/cli"

func rolloutCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "rollout",
		Description: cli.Description{
			Short: "Manage the rollout of a resource",
			Long:  "Manage the rollout of one or many resources.\n        \n Valid resource types include:\n\n  *  deployments\n  *  daemonsets\n  *  statefulsets",
		},
		Examples: []cli.Example{
			{
				Description: "Rollback to the previous deployment",
				CommandLine: "kubectl rollout undo deployment/abc",
			},
			{
				Description: "Check the rollout status of a daemonset",
				CommandLine: "kubectl rollout status daemonset/foo",
			},
			{
				Description: "Restart a deployment",
				CommandLine: "kubectl rollout restart deployment/abc",
			},
			{
				Description: "Restart deployments with the 'app=nginx' label",
				CommandLine: "kubectl rollout restart deployment --selector=app=nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			cb.Subcommands.Command(rolloutHistoryCommand(client))
			cb.Subcommands.Command(rolloutPauseCommand(client))
			cb.Subcommands.Command(rolloutRestartCommand(client))
			cb.Subcommands.Command(rolloutResumeCommand(client))
			cb.Subcommands.Command(rolloutStatusCommand(client))
			cb.Subcommands.Command(rolloutUndoCommand(client))
		},
	}
	return cmd
}

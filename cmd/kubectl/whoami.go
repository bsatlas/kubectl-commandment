package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type authWhoAmIConfig struct {
	Client clientConfig
	Stdout io.Writer

	AllowMissingTemplateKeys bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	ShowManagedFields        bool
	Template                 string
}

func authWhoAmICommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[authWhoAmIConfig] {
	cmd := cli.Command[authWhoAmIConfig]{
		Name: "whoami",
		Description: cli.Description{
			Short: "Experimental: Check self subject attributes",
			Long:  "Experimental: Check who you are and your attributes (groups, extra).\n\n        This command is helpful to get yourself aware of the current user attributes,\n        especially when dynamic authentication, e.g., token webhook, auth proxy, or OIDC provider,\n        is enabled in the Kubernetes cluster.",
		},
		Examples: []cli.Example{
			{
				Description: "Get your subject attributes",
				CommandLine: "kubectl auth whoami",
			},
			{
				Description: "Get your subject attributes in JSON format",
				CommandLine: "kubectl auth whoami -o json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[authWhoAmIConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)
		},
		Run: runAuthWhoAmI,
	}
	return cmd
}

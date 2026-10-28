package main

import "github.com/abstractionsinc/commandment/cli"

type setConfig struct {
	Arguments []string
}

func setCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[setConfig] {
	cmd := cli.Command[setConfig]{
		Name: "set",
		Description: cli.Description{
			Short: "Set specific features on objects",
			Long:  "Configure application resources.\n\n These commands help you make changes to existing application resources.",
		},
		Declare: func(cb *cli.CommandBuilder[setConfig]) {
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[setConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore())
			})

			env := setEnvCommand(client)
			cb.Subcommands.Command(env)
			image := setImageCommand(client)
			cb.Subcommands.Command(image)
			resources := setResourcesCommand(client)
			cb.Subcommands.Command(resources)
			selector := setSelectorCommand(client)
			cb.Subcommands.Command(selector)
			serviceAccount := setServiceAccountCommand(client)
			cb.Subcommands.Command(serviceAccount)
			subject := setSubjectCommand(client)
			cb.Subcommands.Command(subject)
		},
	}
	return cmd
}

package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type cordonConfig struct {
	Client    clientConfig
	Streams   genericiooptions.IOStreams
	Arguments []string
	DryRun    string
	Selector  string
}

func cordonCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[cordonConfig] {
	cmd := cli.Command[cordonConfig]{
		Name: "cordon",
		Description: cli.Description{
			Short: "Mark node as unschedulable",
			Long:  "Mark node as unschedulable.",
		},
		Examples: []cli.Example{
			{
				Description: "Mark node \"foo\" as unschedulable",
				CommandLine: "kubectl cordon foo",
			},
		},
		Declare: func(cb *cli.CommandBuilder[cordonConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[cordonConfig]) {
				form.Sequence(arguments, "NODE", cli.Count.ZeroOrMore()).
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *cordonConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						"node",
					)).
					From(client)
			})

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlag.Long("dry-run").ImplicitValue("unchanged")

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')
		},
		Run: runCordon,
	}
	return cmd
}

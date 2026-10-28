package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type uncordonConfig struct {
	Client    clientConfig
	Streams   genericiooptions.IOStreams
	Arguments []string
	DryRun    string
	Selector  string
}

func uncordonCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[uncordonConfig] {
	cmd := cli.Command[uncordonConfig]{
		Name: "uncordon",
		Description: cli.Description{
			Short: "Mark node as schedulable",
			Long:  "Mark node as schedulable.",
		},
		Examples: []cli.Example{
			{
				Description: "Mark node \"foo\" as schedulable",
				CommandLine: "kubectl uncordon foo",
			},
		},
		Declare: func(cb *cli.CommandBuilder[uncordonConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[uncordonConfig]) {
				form.Sequence(arguments, "NODE", cli.Count.ZeroOrMore()).
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *uncordonConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
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
		Run: runUncordon,
	}
	return cmd
}

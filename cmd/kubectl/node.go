package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type topNodeConfig struct {
	Client             clientConfig
	Streams            genericiooptions.IOStreams
	Arguments          []string
	NoHeaders          bool
	Selector           string
	ShowCapacity       bool
	ShowSwap           bool
	SortBy             string
	UseProtocolBuffers bool
}

func topNodeCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[topNodeConfig] {
	cmd := cli.Command[topNodeConfig]{
		Name:    "node",
		Aliases: cli.Aliases("nodes", "no"),
		Description: cli.Description{
			Short: "Display resource (CPU/memory) usage of nodes",
			Long:  "Display resource (CPU/memory) usage of nodes.\n\n The top-node command allows you to see the resource consumption of nodes.",
		},
		Examples: []cli.Example{
			{
				Description: "Show metrics for all nodes",
				CommandLine: "kubectl top node",
			},
			{
				Description: "Show metrics for a given node",
				CommandLine: "kubectl top node NODE_NAME",
			},
		},
		Declare: func(cb *cli.CommandBuilder[topNodeConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[topNodeConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *topNodeConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						"node",
					)).
					From(client)
			})

			noHeadersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders))
			noHeadersFlag = noHeadersFlag.Description("If present, print output without headers")
			noHeadersFlag.Long("no-headers").ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showCapacityFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowCapacity))
			showCapacityFlag = showCapacityFlag.Description("Print node resources based on Capacity instead of Allocatable(default) of the nodes.")
			showCapacityFlag.Long("show-capacity").ImplicitValue(true)

			showSwapFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowSwap))
			showSwapFlag = showSwapFlag.Description("Print node resources related to swap memory.")
			showSwapFlag.Long("show-swap").ImplicitValue(true)

			sortByFlag := cb.Flags.String(&cb.Config.SortBy)
			sortByFlag = sortByFlag.Description("If non-empty, sort nodes list using specified field. The field can be either 'cpu' or 'memory'.")
			sortByFlag.Long("sort-by")

			useProtocolBuffersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.UseProtocolBuffers))
			useProtocolBuffersFlag = useProtocolBuffersFlag.Description("Enables using protocol-buffers to access Metrics API.")
			cb.Values.Default(useProtocolBuffersFlag, true)
			useProtocolBuffersFlag.Long("use-protocol-buffers").ImplicitValue(true)
		},
		Run: runTopNode,
	}
	return cmd
}

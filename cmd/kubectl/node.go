package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type topNodeConfig struct {
	Client             clientConfig
	Streams            genericiooptions.IOStreams
	Name               string
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

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[topNodeConfig]) {})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[topNodeConfig]) {
				form.ValueFor(name, "NAME").
					CompleteBy(func(ctx context.Context, cfg *topNodeConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						return completeResourceNames(ctx, restClientGetter(cfg.Client), "node", resourceCompletionBare)
					}).
					From(client)
			})

			noHeadersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders)).Description("If present, print output without headers")
			noHeadersFlag.Long("no-headers").ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showCapacityFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowCapacity)).Description("Print node resources based on Capacity instead of Allocatable(default) of the nodes.")
			showCapacityFlag.Long("show-capacity").ImplicitValue(true)

			showSwapFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowSwap)).Description("Print node resources related to swap memory.")
			showSwapFlag.Long("show-swap").ImplicitValue(true)

			sortBy := cb.Values.String(&cb.Config.SortBy)
			sortByFlag := cb.Flags.ValueFor(sortBy).Description("If non-empty, sort nodes list using specified field. The field can be either 'cpu' or 'memory'.")
			sortByFlag.Long("sort-by")
			cb.Rule.In(sortBy, "", "cpu", "memory")

			useProtocolBuffersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.UseProtocolBuffers)).Description("Enables using protocol-buffers to access Metrics API.")
			cb.Values.Default(useProtocolBuffersFlag, true)
			useProtocolBuffersFlag.Long("use-protocol-buffers").ImplicitValue(true)

			cb.Rule.Check("kubectl-top-node-name-selector", func(_ context.Context, cfg *topNodeConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Name == "" || cfg.Selector == "", nil
			}).
				DependsOn(name.OptionalDependency()).
				DependsOn(selectorFlag.Destination().OptionalDependency())
		},
		Run: runTopNode,
	}
	return cmd
}

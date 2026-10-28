package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type topPodConfig struct {
	Client             clientConfig
	Streams            genericiooptions.IOStreams
	Arguments          []string
	AllNamespaces      bool
	Containers         bool
	FieldSelector      string
	NoHeaders          bool
	Selector           string
	ShowSwap           bool
	SortBy             string
	Sum                bool
	UseProtocolBuffers bool
}

func topPodCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[topPodConfig] {
	cmd := cli.Command[topPodConfig]{
		Name:    "pod",
		Aliases: cli.Aliases("pods", "po"),
		Description: cli.Description{
			Short: "Display resource (CPU/memory) usage of pods",
			Long:  "Display resource (CPU/memory) usage of pods.\n\n The 'top pod' command allows you to see the resource consumption of pods.\n\n Due to the metrics pipeline delay, they may be unavailable for a few minutes since pod creation.",
		},
		Examples: []cli.Example{
			{
				Description: "Show metrics for all pods in the default namespace",
				CommandLine: "kubectl top pod",
			},
			{
				Description: "Show metrics for all pods in the given namespace",
				CommandLine: "kubectl top pod --namespace=NAMESPACE",
			},
			{
				Description: "Show metrics for a given pod and its containers",
				CommandLine: "kubectl top pod POD_NAME --containers",
			},
			{
				Description: "Show metrics for the pods defined by label name=myLabel",
				CommandLine: "kubectl top pod -l name=myLabel",
			},
			{
				Description: "Show metrics for all pods on a specific node",
				CommandLine: "kubectl top pod -A --field-selector spec.nodeName=NODE_NAME",
			},
		},
		Declare: func(cb *cli.CommandBuilder[topPodConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[topPodConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *topPodConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						"pod",
					)).
					From(client)
			})

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag = allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			containersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Containers))
			containersFlag = containersFlag.Description("If present, print usage of containers within a pod.")
			containersFlag.Long("containers").ImplicitValue(true)

			fieldSelectorFlag := cb.Flags.String(&cb.Config.FieldSelector)
			fieldSelectorFlag = fieldSelectorFlag.Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			noHeadersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders))
			noHeadersFlag = noHeadersFlag.Description("If present, print output without headers.")
			noHeadersFlag.Long("no-headers").ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showSwapFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowSwap))
			showSwapFlag = showSwapFlag.Description("Print pod resources related to swap memory.")
			showSwapFlag.Long("show-swap").ImplicitValue(true)

			sortByFlag := cb.Flags.String(&cb.Config.SortBy)
			sortByFlag = sortByFlag.Description("If non-empty, sort pods list using specified field. The field can be either 'cpu' or 'memory'.")
			sortByFlag.Long("sort-by")

			sumFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Sum))
			sumFlag = sumFlag.Description("Print the sum of the resource usage")
			sumFlag.Long("sum").ImplicitValue(true)

			useProtocolBuffersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.UseProtocolBuffers))
			useProtocolBuffersFlag = useProtocolBuffersFlag.Description("Enables using protocol-buffers to access Metrics API.")
			cb.Values.Default(useProtocolBuffersFlag, true)
			useProtocolBuffersFlag.Long("use-protocol-buffers").ImplicitValue(true)
		},
		Run: runTopPod,
	}
	return cmd
}

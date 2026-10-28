package kubectl

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

type topPodConfig struct {
	Client             clientConfig
	Streams            genericiooptions.IOStreams
	Name               string
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

func topCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "top",
		Description: cli.Description{
			Short: "Display resource (CPU/memory) usage",
			Long:  "Display resource (CPU/memory) usage.\n\n This command provides a view of recent resource consumption for nodes and pods. It fetches metrics from the Metrics Server, which aggregates this data from the kubelet on each node. The Metrics Server must be installed and running in the cluster for this command to work.\n\n The metrics shown are specifically optimized for Kubernetes autoscaling decisions, such as those made by the Horizontal Pod Autoscaler (HPA) and Vertical Pod Autoscaler (VPA). Because of this, the values may not match those from standard OS tools like 'top', as the metrics are designed to provide a stable signal for autoscalers rather than for pinpoint accuracy.\n\n FAQ for Metrics Server: https://github.com/kubernetes-sigs/metrics-server/blob/master/FAQ.md\n\n When to use this command:\n\n  *  For on-the-fly spot-checks of resource usage (e.g. identify which pods are consuming the most resources at a glance, or get a quick sense of the load on your nodes)\n  *  Understand current resource consumption patterns\n  *  Validate the behavior of your HPA or VPA configurations by seeing the metrics they use for scaling decisions.\n\n It is not intended to be a replacement for full-featured monitoring solutions. Its primary design goal is to provide a low-overhead signal for autoscalers, not to be a perfectly accurate monitoring tool. For high-accuracy reporting, historical analysis, dashboarding, or alerting, you should use a dedicated monitoring solution.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			node := topNodeCommand(client)
			cb.Subcommands.Command(node)
			pod := topPodCommand(client)
			cb.Subcommands.Command(pod)
		},
	}
	return cmd
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

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[topPodConfig]) {})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[topPodConfig]) {
				form.ValueFor(name, "NAME").
					CompleteBy(func(ctx context.Context, cfg *topPodConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
						return completeResourceNames(ctx, restClientGetter(cfg.Client), "pod", resourceCompletionBare)
					}).
					From(client)
			})

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces)).Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			containersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Containers)).Description("If present, print usage of containers within a pod.")
			containersFlag.Long("containers").ImplicitValue(true)

			fieldSelectorFlag := cb.Flags.String(&cb.Config.FieldSelector).Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			noHeadersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders)).Description("If present, print output without headers.")
			noHeadersFlag.Long("no-headers").ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showSwapFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowSwap)).Description("Print pod resources related to swap memory.")
			showSwapFlag.Long("show-swap").ImplicitValue(true)

			sortBy := cb.Values.String(&cb.Config.SortBy)
			sortByFlag := cb.Flags.ValueFor(sortBy).Description("If non-empty, sort pods list using specified field. The field can be either 'cpu' or 'memory'.")
			sortByFlag.Long("sort-by")
			cb.Rule.In(sortBy, "", "cpu", "memory")

			sumFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Sum)).Description("Print the sum of the resource usage")
			sumFlag.Long("sum").ImplicitValue(true)

			useProtocolBuffersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.UseProtocolBuffers)).Description("Enables using protocol-buffers to access Metrics API.")
			cb.Values.Default(useProtocolBuffersFlag, true)
			useProtocolBuffersFlag.Long("use-protocol-buffers").ImplicitValue(true)

			cb.Rule.Check("kubectl-top-pod-name-selector", func(_ context.Context, cfg *topPodConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Name == "" || cfg.Selector == "" && cfg.FieldSelector == "", nil
			}).
				DependsOn(name.OptionalDependency()).
				DependsOn(selectorFlag.Destination().OptionalDependency()).
				DependsOn(fieldSelectorFlag.Destination().OptionalDependency())
		},
		Run: runTopPod,
	}
	return cmd
}

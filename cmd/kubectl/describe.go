package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type describeConfig struct {
	Client          clientConfig
	Streams         genericiooptions.IOStreams
	ShowEventsState cli.OptionalValue[cli.ValueState]
	Arguments       []string
	AllNamespaces   bool
	ChunkSize       int64
	Filename        []string
	Kustomize       string
	Recursive       bool
	Selector        string
	ShowEvents      bool
}

func describeCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[describeConfig] {
	cmd := cli.Command[describeConfig]{
		Name: "describe",
		Description: cli.Description{
			Short: "Show details of a specific resource or group of resources",
			Long:  "Show details of a specific resource or group of resources.\n\n Print a detailed description of the selected resources, including related resources such as events or controllers. You may select a single object by name, all objects of that type, provide a name prefix, or label selector. For example:\n\n        $ kubectl describe TYPE NAME_PREFIX\n        \n will first check for an exact match on TYPE and NAME_PREFIX. If no such resource exists, it will output details for every resource that has a name prefixed with NAME_PREFIX.\n\nUse \"kubectl api-resources\" for a complete list of supported resources.",
		},
		Examples: []cli.Example{
			{
				Description: "Describe a node",
				CommandLine: "kubectl describe nodes kubernetes-node-emt8.c.myproject.internal",
			},
			{
				Description: "Describe a pod",
				CommandLine: "kubectl describe pods/nginx",
			},
			{
				Description: "Describe a pod identified by type and name in \"pod.json\"",
				CommandLine: "kubectl describe -f pod.json",
			},
			{
				Description: "Describe all pods",
				CommandLine: "kubectl describe pods",
			},
			{
				Description: "Describe pods by label name=myLabel",
				CommandLine: "kubectl describe pods -l name=myLabel",
			},
			{
				Description: "Describe all pods managed by the 'frontend' replication controller\n(rc-created pods get the name of the rc as a prefix in the pod name)",
				CommandLine: "kubectl describe pods frontend",
			},
		},
		Declare: func(cb *cli.CommandBuilder[describeConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[describeConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *describeConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag = allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			chunkSizeFlag := cb.Flags.Value(&cb.Config.ChunkSize, int64Language())
			chunkSizeFlag = chunkSizeFlag.Description("Return large lists in chunks rather than all at once. Pass 0 to disable.")
			cb.Values.Default(chunkSizeFlag, int64(500))
			chunkSizeFlag.Long("chunk-size")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files containing the resource to describe")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showEventsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowEvents))
			showEventsFlag = showEventsFlag.Description("If true, display events related to the described object. Defaults to true for a single object, false for multiple objects and prefix matching.")
			cb.Values.Default(showEventsFlag, true)
			showEventsFlag.Long("show-events").ImplicitValue(true)
			cb.Access.ValueState(showEventsFlag, &cb.Config.ShowEventsState)
		},
		Run: runDescribe,
	}
	return cmd
}

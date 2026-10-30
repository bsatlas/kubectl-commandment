package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmddescribe "k8s.io/kubectl/pkg/cmd/describe"
	kubectldescribe "k8s.io/kubectl/pkg/describe"
)

type describeConfig struct {
	Client             clientConfig
	Streams            genericiooptions.IOStreams
	ShowEventsState    cli.OptionalValue[cli.ValueState]
	ResourceType       string
	Names              []string
	QualifiedResources []string
	AllNamespaces      bool
	ChunkSize          int64
	FilenameGroups     [][]string
	Kustomize          string
	Recursive          bool
	Selector           string
	ShowEvents         bool
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

			resourceType := cb.Values.Value(&cb.Config.ResourceType, resourceTypeLanguage)
			names := cb.Values.Sequence(&cb.Config.Names, describeNamePrefixLanguage)
			qualifiedResources := cb.Values.Sequence(&cb.Config.QualifiedResources, qualifiedResourceLanguage)

			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[describeConfig]) {})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[describeConfig]) {
				form.ValueFor(resourceType, "TYPE[.VERSION][.GROUP]").
					CompleteBy(completeDescribeResourceType).
					From(client)
				form.Sequence(names, "NAME_PREFIX", cli.Count.ZeroOrMore()).
					CompleteBy(completeDescribeResourceName(resourceType)).
					From(client, resourceType.OptionalDependency())
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[describeConfig]) {
				form.Sequence(qualifiedResources, "TYPE[.VERSION][.GROUP]/NAME", cli.Count.OneOrMore()).
					CompleteBy(completeDescribeQualifiedResource).
					From(client)
			})

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces)).Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			chunkSizeFlag := cb.Flags.Value(&cb.Config.ChunkSize, int64Language()).Description("Return large lists in chunks rather than all at once. Pass 0 to disable.")
			cb.Values.Default(chunkSizeFlag, int64(500))
			chunkSizeFlag.Long("chunk-size")

			filenameGroups := cb.Values.Sequence(&cb.Config.FilenameGroups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameGroups).
				Description("Filename, directory, or URL to files containing the resource to describe").
				CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize).
				Description("Process the kustomization directory. This flag can't be used together with -f or -R.").
				CompleteWith(cli.CompletionNativeDirectories)
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive)).Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			showEventsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowEvents)).Description("If true, display events related to the described object. Defaults to true for a single object, false for multiple objects and prefix matching.")
			cb.Values.Default(showEventsFlag, true)
			showEventsFlag.Long("show-events").ImplicitValue(true)
			cb.Access.ValueState(showEventsFlag, &cb.Config.ShowEventsState)

			cb.Rule.Check("kubectl-describe-input-source", func(_ context.Context, cfg *describeConfig, _ cli.Dependencies) (bool, error) {
				return cfg.ResourceType != "" || len(cfg.QualifiedResources) > 0 || len(slices.Concat(cfg.FilenameGroups...)) > 0 || cfg.Kustomize != "", nil
			}).
				DependsOn(resourceType.OptionalDependency()).
				DependsOn(qualifiedResources.OptionalDependency()).
				DependsOn(filenameGroups.OptionalDependency()).
				DependsOn(kustomizeFlag.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-describe-filename-source", func(_ context.Context, cfg *describeConfig, _ cli.Dependencies) (bool, error) {
				return len(slices.Concat(cfg.FilenameGroups...)) == 0 || cfg.Kustomize == "", nil
			}).
				DependsOn(filenameGroups.OptionalDependency()).
				DependsOn(kustomizeFlag.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-describe-kustomize-recursive", func(_ context.Context, cfg *describeConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Kustomize == "" || !cfg.Recursive, nil
			}).
				DependsOn(kustomizeFlag.Destination().OptionalDependency()).
				DependsOn(recursiveFlag.Destination().OptionalDependency())
		},
		Run: runDescribe,
	}
	return cmd
}

func completeDescribeResourceType(_ context.Context, cfg *describeConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	return completeResourceTypes(restClientGetter(cfg.Client), resourceCompletionBare, request.Text)
}

func completeDescribeResourceName(resourceType cli.Destination[describeConfig, string]) cli.ValueCompleter[describeConfig] {
	return func(ctx context.Context, cfg *describeConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		state, ok := cli.ValueStateFromContext(ctx, resourceType)
		if !ok || state.State() != cli.ResolutionStateResolved || cfg.ResourceType == "" {
			return cli.CompletionContribution{}, nil
		}
		return completeResourceNames(ctx, restClientGetter(cfg.Client), cfg.ResourceType, resourceCompletionBare)
	}
}

func completeDescribeQualifiedResource(ctx context.Context, cfg *describeConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	resourceType, _, found := strings.Cut(request.Text, "/")
	if !found {
		return completeResourceTypes(restClientGetter(cfg.Client), resourceCompletionQualified, request.Text)
	}
	return completeResourceNames(ctx, restClientGetter(cfg.Client), resourceType, resourceCompletionQualified)
}

var describeNamePrefixLanguage = cli.ValueCodec[string]{
	Name: "kubectl-describe-name-prefix",
	Parse: func(value string) (string, error) {
		if strings.Contains(value, "/") {
			return "", cli.Rejection.Message(
				"name-prefix",
				fmt.Sprintf("expected name prefix, got %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

func runDescribe(_ context.Context, cfg *describeConfig) error {
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	if cfg.AllNamespaces {
		enforceNamespace = false
	}

	builderArguments := slices.Clone(cfg.QualifiedResources)
	if cfg.ResourceType != "" {
		builderArguments = make([]string, 0, 1+len(cfg.Names))
		builderArguments = append(builderArguments, cfg.ResourceType)
		builderArguments = append(builderArguments, cfg.Names...)
	}

	filenameOptions := resource.FilenameOptions{
		Filenames: slices.Concat(cfg.FilenameGroups...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	describerSettings := &kubectldescribe.DescriberSettings{
		ShowEvents: cfg.ShowEvents,
		ChunkSize:  cfg.ChunkSize,
	}
	options := cmddescribe.DescribeOptions{
		Selector:  cfg.Selector,
		Namespace: namespace,
		Describer: func(mapping *meta.RESTMapping) (kubectldescribe.ResourceDescriber, error) {
			return kubectldescribe.DescriberFn(factory, mapping)
		},
		NewBuilder:              factory.NewBuilder,
		BuilderArgs:             builderArguments,
		EnforceNamespace:        enforceNamespace,
		AllNamespaces:           cfg.AllNamespaces,
		ShowEventsExplicitlySet: explicitFlagValue(cfg.ShowEventsState),
		DescriberSettings:       describerSettings,
		FilenameOptions:         &filenameOptions,
		IOStreams:               cfg.Streams,
	}
	return options.Run()
}

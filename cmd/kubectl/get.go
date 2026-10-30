package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

var getOutputFormats = [...]string{
	"json",
	"yaml",
	"kyaml",
	"name",
	"go-template",
	"go-template-file",
	"template",
	"templatefile",
	"jsonpath",
	"jsonpath-as-json",
	"jsonpath-file",
	"custom-columns",
	"custom-columns-file",
	"wide",
}

type getConfig struct {
	Client clientConfig

	ResourceType       string
	Names              []string
	QualifiedResources []string

	Output                   string
	NoHeaders                bool
	ShowManagedFields        bool
	Template                 string
	AllowMissingTemplateKeys bool
	ShowLabels               bool
	SortBy                   string
	LabelColumns             []string
	ShowKind                 bool

	Raw                  string
	Watch                bool
	WatchOnly            bool
	OutputWatchEvents    bool
	IgnoreNotFound       bool
	FieldSelector        string
	AllNamespaces        bool
	ServerPrintRequested bool
	ServerPrint          bool
	HumanReadable        bool
	FilenameGroups       [][]string
	Kustomize            string
	Recursive            bool
	ChunkSize            int64
	LabelSelector        string
	Subresource          string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func getCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[getConfig] {
	cmd := cli.Command[getConfig]{
		Name: "get",
		Description: cli.Description{
			Short: "Display one or many resources",
			Long:  "Display one or many resources.\n\n Prints a table of the most important information about the specified resources. You can filter the list using a label selector and the --selector flag. If the desired resource type is namespaced you will only see results in the current namespace if you don't specify any namespace.\n\n By specifying the output as 'template' and providing a Go template as the value of the --template flag, you can filter the attributes of the fetched resources.\n\nUse \"kubectl api-resources\" for a complete list of supported resources.",
		},
		Examples: []cli.Example{
			{
				Description: "List all pods in ps output format",
				CommandLine: "kubectl get pods",
			},
			{
				Description: "List all pods in ps output format with more information (such as node name)",
				CommandLine: "kubectl get pods -o wide",
			},
			{
				Description: "List a single replication controller with specified NAME in ps output format",
				CommandLine: "kubectl get replicationcontroller web",
			},
			{
				Description: "List deployments in JSON output format, in the \"v1\" version of the \"apps\" API group",
				CommandLine: "kubectl get deployments.v1.apps -o json",
			},
			{
				Description: "List a single pod in JSON output format",
				CommandLine: "kubectl get -o json pod web-pod-13je7",
			},
			{
				Description: "List a pod identified by type and name specified in \"pod.yaml\" in JSON output format",
				CommandLine: "kubectl get -f pod.yaml -o json",
			},
			{
				Description: "List resources from a directory with kustomization.yaml - e.g. dir/kustomization.yaml",
				CommandLine: "kubectl get -k dir/",
			},
			{
				Description: "Return only the phase value of the specified pod",
				CommandLine: "kubectl get -o template pod/web-pod-13je7 --template={{.status.phase}}",
			},
			{
				Description: "List resource information in custom columns",
				CommandLine: "kubectl get pod test-pod -o custom-columns=CONTAINER:.spec.containers[0].name,IMAGE:.spec.containers[0].image",
			},
			{
				Description: "List all replication controllers and services together in ps output format",
				CommandLine: "kubectl get rc,services",
			},
			{
				Description: "List one or more resources by their type and names",
				CommandLine: "kubectl get rc/web service/frontend pods/web-pod-13je7",
			},
			{
				Description: "List the 'status' subresource for a single pod",
				CommandLine: "kubectl get pod web-pod-13je7 --subresource status",
			},
			{
				Description: "List all deployments in namespace 'backend'",
				CommandLine: "kubectl get deployments.apps --namespace backend",
			},
			{
				Description: "List all pods existing in all namespaces",
				CommandLine: "kubectl get pods --all-namespaces",
			},
		},
		Declare: func(cb *cli.CommandBuilder[getConfig]) {
			cb.Access.Stdin(&cb.Config.Stdin)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			client := deriveClientConfig(cb, &cb.Config.Client, client)

			resourceType := cb.Values.Value(&cb.Config.ResourceType, resourceTypeLanguage)
			names := cb.Values.Sequence(&cb.Config.Names, resourceNameLanguage)
			qualifiedResources := cb.Values.Sequence(&cb.Config.QualifiedResources, qualifiedResourceLanguage)

			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[getConfig]) {})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[getConfig]) {
				form.ValueFor(resourceType, "TYPE[.VERSION][.GROUP]").
					CompleteBy(completeGetResourceType).
					From(client)
				form.Sequence(names, "NAME", cli.Count.ZeroOrMore()).
					CompleteBy(completeGetResourceName(resourceType)).
					From(client, resourceType.OptionalDependency())
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[getConfig]) {
				form.Sequence(qualifiedResources, "TYPE[.VERSION][.GROUP]/NAME", cli.Count.OneOrMore()).
					CompleteBy(completeGetQualifiedResource).
					From(client)
			})
			output := cb.Values.Value(&cb.Config.Output, getOutputLanguage)
			cb.Values.Default(output, "")
			outputFlag := cb.Flags.ValueFor(output).
				ValueName("FORMAT").
				Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file, custom-columns, custom-columns-file, wide). See custom columns [https://kubernetes.io/docs/reference/kubectl/#custom-columns], golang template [http://golang.org/pkg/text/template/#pkg-overview] and jsonpath template [https://kubernetes.io/docs/reference/kubectl/jsonpath/].").
				CompleteBy(literalCompleter[getConfig](cli.CompletionCandidateValue, getOutputFormats[:]...))
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			noHeaders := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders)).Description("When using the default or custom-column output format, don't print headers (default print headers).")
			noHeaders.Long("no-headers").ImplicitValue(true)

			showManagedFields := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields)).Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFields.Long("show-managed-fields").ImplicitValue(true)

			template := cb.Flags.String(&cb.Config.Template).ValueName("TEMPLATE").Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").CompleteWith(cli.CompletionNativeFiles)
			template.Long("template")

			allowMissingTemplateKeys := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissingTemplateKeys, true)
			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(allowMissingTemplateKeys).ValueName("BOOL").Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			showLabels := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowLabels)).Description("When printing, show all labels as the last column (default hide labels column)")
			showLabels.Long("show-labels").ImplicitValue(true)

			sortBy := cb.Flags.String(&cb.Config.SortBy).ValueName("JSONPATH").Description("If non-empty, sort list types using this field specification. The field specification is expressed as a JSONPath expression (e.g. '{.metadata.name}'). The field in the API resource specified by this JSONPath expression must be an integer or a string.")
			sortBy.Long("sort-by")

			labelColumns := cb.Values.Sequence(&cb.Config.LabelColumns, cli.Value.String())
			labelColumnsFlag := cb.Flags.Sequence(labelColumns).
				DelimitedBy(",").
				ValueName("LABEL").
				Description("Accepts a comma separated list of labels that are going to be presented as columns. Names are case-sensitive. You can also use multiple flag options like -L label1 -L label2...")
			labelColumnsFlag.Long("label-columns")
			labelColumnsFlag.Short('L').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			showKind := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowKind)).Description("If present, list the resource type for the requested object(s).")
			showKind.Long("show-kind").ImplicitValue(true)

			raw := cb.Flags.String(&cb.Config.Raw).ValueName("URI").Description("Raw URI to request from the server. Uses the transport specified by the kubeconfig file.")
			raw.Long("raw")

			watch := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Watch)).Description("After listing/getting the requested object, watch for changes.")
			watch.Long("watch").ImplicitValue(true)
			watch.Short('w').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			watchOnly := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.WatchOnly)).Description("Watch for changes to the requested object(s), without listing/getting first.")
			watchOnly.Long("watch-only").ImplicitValue(true)

			outputWatchEvents := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.OutputWatchEvents)).Description("Output watch event objects when --watch or --watch-only is used. Existing objects are output as initial ADDED events.")
			outputWatchEvents.Long("output-watch-events").ImplicitValue(true)

			ignoreNotFound := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.IgnoreNotFound)).Description("If set to true, suppresses NotFound error for specific objects that do not exist. Using this flag with commands that query for collections of resources has no effect when no resources are found.")
			ignoreNotFound.Long("ignore-not-found").ImplicitValue(true)

			fieldSelector := cb.Flags.String(&cb.Config.FieldSelector).ValueName("SELECTOR").Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelector.Long("field-selector")

			allNamespaces := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces)).Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespaces.Long("all-namespaces").ImplicitValue(true)
			allNamespaces.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			serverPrintRequested := cb.Values.Bool(&cb.Config.ServerPrintRequested)
			cb.Values.Default(serverPrintRequested, true)
			serverPrintFlag := cb.Flags.ValueFor(serverPrintRequested).ValueName("BOOL").Description("If true, have the server return the appropriate table output. Supports extension APIs and CRDs.")
			serverPrintFlag.Long("server-print").ImplicitValue(true)

			cb.Values.Derive(&cb.Config.ServerPrint, cli.Value.ComputedBy(func(_ context.Context, _ *getConfig, dependencies cli.Dependencies) (bool, error) {
				requested, ok := dependencies.Get(serverPrintRequested)
				if !ok {
					return false, fmt.Errorf("server-print dependency is unavailable")
				}
				format, ok := dependencies.Get(output)
				if !ok {
					return false, fmt.Errorf("output dependency is unavailable")
				}
				if strings.Contains(format, "custom-columns") || format == "yaml" || strings.Contains(format, "json") {
					return false, nil
				}
				return requested, nil
			}).From(serverPrintRequested, output))

			cb.Values.Derive(&cb.Config.HumanReadable, cli.Value.ComputedBy(func(_ context.Context, _ *getConfig, dependencies cli.Dependencies) (bool, error) {
				format, ok := dependencies.Get(output)
				if !ok {
					return false, fmt.Errorf("output dependency is unavailable")
				}
				templateValue, _ := dependencies.Get(template)
				return format == "" && templateValue == "" || format == "wide", nil
			}).From(output, template.OptionalDependency()))

			filenameGroups := cb.Values.Sequence(&cb.Config.FilenameGroups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameGroups).
				ValueName("FILENAME").
				Description("Filename, directory, or URL to files identifying the resource to get from a server.").
				CompleteWith(cli.CompletionNativePaths)
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomize := cb.Flags.String(&cb.Config.Kustomize).
				ValueName("DIR").
				Description("Process the kustomization directory. This flag can't be used together with -f or -R.").
				CompleteWith(cli.CompletionNativeDirectories)
			kustomize.Long("kustomize")
			kustomize.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			recursive := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive)).Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursive.Long("recursive").ImplicitValue(true)
			recursive.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			chunkSize := cb.Values.Value(&cb.Config.ChunkSize, int64Language())
			cb.Values.Default(chunkSize, 500)
			chunkSizeFlag := cb.Flags.ValueFor(chunkSize).ValueName("SIZE").Description("Return large lists in chunks rather than all at once. Pass 0 to disable.")
			chunkSizeFlag.Long("chunk-size")

			labelSelector := cb.Flags.String(&cb.Config.LabelSelector).ValueName("SELECTOR").Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			labelSelector.Long("selector")
			labelSelector.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			subresource := cb.Flags.String(&cb.Config.Subresource).ValueName("SUBRESOURCE").Description("If specified, gets the subresource of the requested object.")
			subresource.CompleteBy(literalCompleter[getConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
			subresource.Long("subresource")

			rawSelected := cb.When.NotEqual(raw.Destination(), "")
			cb.Rule.When(rawSelected).Forbid(resourceType)
			cb.Rule.When(rawSelected).Forbid(names)
			cb.Rule.When(rawSelected).Forbid(qualifiedResources)

			cb.Rule.When(cb.When.Equal(showLabels.Destination(), true)).In(output, "", "wide")

			cb.Rule.Check("kubectl-get-raw-options", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw == "" || !cfg.Watch && !cfg.WatchOnly && cfg.LabelSelector == "" && cfg.Output == "", nil
			}).
				DependsOn(raw.Destination().OptionalDependency()).
				DependsOn(watch.Destination().OptionalDependency()).
				DependsOn(watchOnly.Destination().OptionalDependency()).
				DependsOn(labelSelector.Destination().OptionalDependency()).
				DependsOn(output.OptionalDependency())

			cb.Rule.Check("kubectl-get-output-watch-events", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.OutputWatchEvents || cfg.Watch || cfg.WatchOnly, nil
			}).
				DependsOn(outputWatchEvents.Destination().OptionalDependency()).
				DependsOn(watch.Destination().OptionalDependency()).
				DependsOn(watchOnly.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-output-template", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				if strings.Contains(cfg.Output, "=") {
					return cfg.Template == "", nil
				}
				return !getOutputAcceptsArgument(cfg.Output) || cfg.Template != "", nil
			}).
				DependsOn(output.OptionalDependency()).
				DependsOn(template.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-input-source", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Watch || cfg.WatchOnly || cfg.ResourceType != "" || len(cfg.QualifiedResources) > 0 || len(slices.Concat(cfg.FilenameGroups...)) > 0 || cfg.Kustomize != "" || cfg.Raw != "", nil
			}).
				DependsOn(resourceType.OptionalDependency()).
				DependsOn(qualifiedResources.OptionalDependency()).
				DependsOn(filenameGroups.OptionalDependency()).
				DependsOn(kustomize.Destination().OptionalDependency()).
				DependsOn(raw.Destination().OptionalDependency()).
				DependsOn(watch.Destination().OptionalDependency()).
				DependsOn(watchOnly.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-filename-source", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw != "" || len(slices.Concat(cfg.FilenameGroups...)) == 0 || cfg.Kustomize == "", nil
			}).
				DependsOn(raw.Destination().OptionalDependency()).
				DependsOn(filenameGroups.OptionalDependency()).
				DependsOn(kustomize.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-kustomize-recursive", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw != "" || cfg.Kustomize == "" || !cfg.Recursive, nil
			}).
				DependsOn(raw.Destination().OptionalDependency()).
				DependsOn(kustomize.Destination().OptionalDependency()).
				DependsOn(recursive.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-raw-uri", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" {
					return true, nil
				}
				_, err := url.ParseRequestURI(cfg.Raw)
				return err == nil, nil
			}).DependsOn(raw.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-label-columns-output", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.LabelColumns) == 0 || !strings.HasPrefix(cfg.Output, "custom-columns"), nil
			}).
				DependsOn(labelColumns.OptionalDependency()).
				DependsOn(output.OptionalDependency())
		},
		Run: runGet,
	}
	return cmd
}

func getOutputAcceptsArgument(format string) bool {
	switch format {
	case "go-template", "go-template-file", "template", "templatefile", "jsonpath", "jsonpath-as-json", "jsonpath-file", "custom-columns", "custom-columns-file":
		return true
	default:
		return false
	}
}

var getOutputLanguage = cli.ValueCodec[string]{
	Name: "kubectl-output-format",
	Parse: func(value string) (string, error) {
		if value == "" {
			return value, nil
		}
		if strings.EqualFold(value, "json") || strings.EqualFold(value, "yaml") || strings.EqualFold(value, "kyaml") || strings.EqualFold(value, "name") {
			return value, nil
		}
		for _, format := range getOutputFormats[3:] {
			if value == format {
				return value, nil
			}
		}
		format, payload, found := strings.Cut(value, "=")
		if found && payload != "" && getOutputAcceptsArgument(format) {
			return value, nil
		}
		return "", cli.Rejection.Message(
			"output-format",
			fmt.Sprintf("unsupported output format %q", value),
			cli.Rejection.At(0, 0, len(value)),
		)
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

func completeGetResourceType(_ context.Context, cfg *getConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	return completeResourceTypes(restClientGetter(cfg.Client), resourceCompletionBare, request.Text)
}

func completeGetResourceName(resourceType cli.Destination[getConfig, string]) cli.ValueCompleter[getConfig] {
	return func(ctx context.Context, cfg *getConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		state, ok := cli.ValueStateFromContext(ctx, resourceType)
		if !ok || state.State() != cli.ResolutionStateResolved || cfg.ResourceType == "" {
			return cli.CompletionContribution{}, nil
		}
		return completeResourceNames(ctx, restClientGetter(cfg.Client), cfg.ResourceType, resourceCompletionBare)
	}
}

func completeGetQualifiedResource(ctx context.Context, cfg *getConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	resourceType, _, found := strings.Cut(request.Text, "/")
	if !found {
		return completeResourceTypes(restClientGetter(cfg.Client), resourceCompletionQualified, request.Text)
	}
	return completeResourceNames(ctx, restClientGetter(cfg.Client), resourceType, resourceCompletionQualified)
}

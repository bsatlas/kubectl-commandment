package main

import (
	"context"
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/annotate"
	"k8s.io/kubectl/pkg/cmd/util"
)

type annotateOperands struct {
	Resources []string
	New       map[string]string
	Remove    []string
}

func parseAnnotateOperands(arguments []string) (annotateOperands, error) {
	resources, pairs, err := util.GetResourcesAndPairs(arguments, "annotation")
	if err != nil {
		return annotateOperands{}, err
	}
	newAnnotations, removeAnnotations, err := util.ParsePairs(pairs, "annotation", true)
	if err != nil {
		return annotateOperands{}, err
	}
	for _, key := range removeAnnotations {
		if _, ok := newAnnotations[key]; ok {
			return annotateOperands{}, fmt.Errorf("can not both modify and remove annotation %q in the same command", key)
		}
	}
	input := annotateOperands{
		Resources: resources,
		New:       newAnnotations,
		Remove:    removeAnnotations,
	}
	return input, nil
}

type annotateConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	Operands                 annotateOperands
	All                      bool
	AllNamespaces            bool
	AllowMissingTemplateKeys bool
	DryRun                   util.DryRunStrategy
	FieldManager             string
	FieldSelector            string
	Filename                 filenameOptions
	List                     bool
	Local                    bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	Overwrite                bool
	Record                   bool
	ResourceVersion          string
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func annotateCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[annotateConfig] {
	cmd := cli.Command[annotateConfig]{
		Name: "annotate",
		Description: cli.Description{
			Short: "Update the annotations on a resource",
			Long:  "Update the annotations on one or more resources.\n\n All Kubernetes objects support the ability to store additional data with the object as annotations. Annotations are key/value pairs that can be larger than labels and include arbitrary string values such as structured JSON. Tools and system extensions may use annotations to store their own data.\n\n Attempting to set an annotation that already exists will fail unless --overwrite is set. If --resource-version is specified and does not match the current resource version on the server the command will fail.\n\nUse \"kubectl api-resources\" for a complete list of supported resources.",
		},
		Examples: []cli.Example{
			{
				Description: "Update pod 'foo' with the annotation 'description' and the value 'my frontend'\nIf the same annotation is set multiple times, only the last value will be applied",
				CommandLine: "kubectl annotate pods foo description='my frontend'",
			},
			{
				Description: "Update a pod identified by type and name in \"pod.json\"",
				CommandLine: "kubectl annotate -f pod.json description='my frontend'",
			},
			{
				Description: "Update pod 'foo' with the annotation 'description' and the value 'my frontend running nginx', overwriting any existing value",
				CommandLine: "kubectl annotate --overwrite pods foo description='my frontend running nginx'",
			},
			{
				Description: "Update all pods in the namespace",
				CommandLine: "kubectl annotate pods --all description='my frontend running nginx'",
			},
			{
				Description: "Update pod 'foo' only if the resource is unchanged from version 1",
				CommandLine: "kubectl annotate pods foo description='my frontend running nginx' --resource-version=1",
			},
			{
				Description: "Update pod 'foo' by removing an annotation named 'description' if it exists\nDoes not require the --overwrite flag",
				CommandLine: "kubectl annotate pods foo description-",
			},
		},
		Declare: func(cb *cli.CommandBuilder[annotateConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[annotateConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *annotateConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})
			input := cb.Values.Derive(&cb.Config.Operands, cli.Value.ComputedBy(func(_ context.Context, _ *annotateConfig, dependencies cli.Dependencies) (annotateOperands, error) {
				values, ok := dependencies.Get(arguments)
				if !ok {
					return annotateOperands{}, fmt.Errorf("annotate arguments dependency is unavailable")
				}
				return parseAnnotateOperands(values)
			}).From(arguments))

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Select all resources in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			cb.Values.ResolveWith(allNamespacesFlag, cli.Value.KeepLast())
			allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			cb.Values.ResolveWith(allowMissingTemplateKeysFlag, cli.Value.KeepLast())
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRun, util.DryRunNone)
			dryRunFlag.Long("dry-run").ImplicitValue(util.DryRunClient)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-annotate")
			fieldManagerFlag.Long("field-manager")

			fieldSelector := cb.Values.String(&cb.Config.FieldSelector)
			fieldSelectorFlag := cb.Flags.ValueFor(fieldSelector)
			cb.Values.ResolveWith(fieldSelectorFlag, cli.Value.KeepLast())
			fieldSelectorFlag.Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("identifying the resource.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			list := cb.Values.Bool(&cb.Config.List)
			listFlag := cb.Flags.ValueFor(list)
			cb.Values.ResolveWith(listFlag, cli.Value.KeepLast())
			listFlag.Description("If true, display the annotations for a given resource.")
			listFlag.Long("list").ImplicitValue(true)

			local := cb.Values.Bool(&cb.Config.Local)
			localFlag := cb.Flags.ValueFor(local)
			cb.Values.ResolveWith(localFlag, cli.Value.KeepLast())
			localFlag.Description("If true, annotation will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			output := cb.Values.Value(&cb.Config.Output, printOutputLanguage)
			outputFlag := cb.Flags.ValueFor(output)
			cb.Values.ResolveWith(outputFlag, cli.Value.KeepLast())
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.CompleteBy(completePrintOutputFormat[annotateConfig])
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			cb.Access.ValueState(output, &cb.Config.OutputState)

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			cb.Values.ResolveWith(overwriteFlag, cli.Value.KeepLast())
			overwriteFlag.Description("If true, allow annotations to be overwritten, otherwise reject annotation updates that overwrite existing annotations.")
			overwriteFlag.Long("overwrite").ImplicitValue(true)

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			cb.Values.ResolveWith(recordFlag, cli.Value.KeepLast())
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			cb.Values.Default(recursiveFlag, true)
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			resourceVersionFlag := cb.Flags.String(&cb.Config.ResourceVersion)
			cb.Values.ResolveWith(resourceVersionFlag, cli.Value.KeepLast())
			resourceVersionFlag.Description("If non-empty, the annotation update will only succeed if this is the current resource-version for the object. Only valid when specifying a single resource.")
			resourceVersionFlag.Long("resource-version")

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			listing := cb.When.Equal(list, true)
			cb.Rule.When(listing).Equal(output, "")
			selectAll := cb.When.Equal(all, true)
			cb.Rule.When(selectAll).Equal(selector, "")
			cb.Rule.When(selectAll).Equal(fieldSelector, "")
			localOnly := cb.When.Equal(local, true)
			cb.Rule.When(localOnly).NotEqual(dryRun, util.DryRunServer)
			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.Check("kubectl-annotate-source", func(_ context.Context, cfg *annotateConfig, _ cli.Dependencies) (bool, error) {
				hasFilename := len(cfg.Filename.Filenames()) > 0
				hasKustomize := cfg.Filename.Kustomize != ""
				if hasFilename && hasKustomize {
					return false, nil
				}
				hasFiles := hasFilename || hasKustomize
				if cfg.Local {
					return len(cfg.Operands.Resources) == 0 && hasFiles, nil
				}
				return len(cfg.Operands.Resources) > 0 || hasFiles, nil
			}).
				DependsOn(input.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency()).
				DependsOn(local.OptionalDependency())
			cb.Rule.Check("kubectl-annotate-updates", func(_ context.Context, cfg *annotateConfig, _ cli.Dependencies) (bool, error) {
				updates := len(cfg.Operands.New) + len(cfg.Operands.Remove)
				if cfg.List {
					return updates == 0, nil
				}
				return updates > 0, nil
			}).
				DependsOn(input.OptionalDependency()).
				DependsOn(list.OptionalDependency())

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			cb.Values.ResolveWith(showManagedFieldsFlag, cli.Value.KeepLast())
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			cb.Values.ResolveWith(templateFlag, cli.Value.KeepLast())
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.CompleteWith(cli.CompletionNativeFiles)
			templateFlag.Long("template")
			declarePrintOutputTemplateRule(cb, outputFlag, templateFlag)
		},
		Run: runAnnotate,
	}
	return cmd
}

func runAnnotate(_ context.Context, cfg *annotateConfig) error {
	strategy := cfg.DryRun

	factory := cfg.Client.factory()
	flags := annotate.NewAnnotateFlags(factory, cfg.Streams)
	flags.FieldManager = cfg.FieldManager
	flags.List = cfg.List
	flags.Overwrite = cfg.Overwrite
	flags.ResourceVersion = cfg.ResourceVersion
	*flags.ResourceBuilderFlags.LabelSelector = cfg.Selector
	*flags.ResourceBuilderFlags.FieldSelector = cfg.FieldSelector
	*flags.ResourceBuilderFlags.AllNamespaces = cfg.AllNamespaces
	*flags.ResourceBuilderFlags.All = cfg.All
	*flags.ResourceBuilderFlags.Local = cfg.Local
	filenames := cfg.Filename.Filenames()
	flags.ResourceBuilderFlags.FileNameFlags.Filenames = &filenames
	flags.ResourceBuilderFlags.FileNameFlags.Kustomize = &cfg.Filename.Kustomize
	flags.ResourceBuilderFlags.FileNameFlags.Recursive = &cfg.Filename.Recursive
	configurePrintFlagsWithOutputState(flags.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	util.PrintFlagsWithDryRunStrategy(flags.PrintFlags, strategy)
	printer, err := flags.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if err := recordFlags.CompleteWithChangeCause("kubectl annotate"); err != nil {
		return err
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	options := annotate.AnnotateOptions{
		PrintObj:          printer.PrintObj,
		Recorder:          recorder,
		ResourceBuilder:   flags.ResourceBuilderFlags.ToBuilder(factory, cfg.Operands.Resources),
		DryRunStrategy:    strategy,
		FieldManager:      cfg.FieldManager,
		List:              cfg.List,
		Local:             cfg.Local,
		NewAnnotations:    cfg.Operands.New,
		Overwrite:         cfg.Overwrite,
		RemoveAnnotations: cfg.Operands.Remove,
		ResourceVersion:   cfg.ResourceVersion,
		IOStreams:         cfg.Streams,
	}
	return options.RunAnnotate()
}

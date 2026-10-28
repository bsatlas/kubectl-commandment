package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type labelConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	All                      bool
	AllNamespaces            bool
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	FieldSelector            string
	Filename                 []string
	Kustomize                string
	List                     bool
	Local                    bool
	Output                   string
	Overwrite                bool
	Record                   bool
	Recursive                bool
	ResourceVersion          string
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func labelCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[labelConfig] {
	cmd := cli.Command[labelConfig]{
		Name: "label",
		Description: cli.Description{
			Short: "Update the labels on a resource",
			Long:  "Update the labels on a resource.\n\n  *  A label key and value must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores, up to 63 characters each.\n  *  Optionally, the key can begin with a DNS subdomain prefix and a single '/', like example.com/my-app.\n  *  If --overwrite is true, then existing labels can be overwritten, otherwise attempting to overwrite a label will result in an error.\n  *  If --resource-version is specified, then updates will use this resource version, otherwise the existing resource-version will be used.",
		},
		Examples: []cli.Example{
			{
				Description: "Update pod 'foo' with the label 'unhealthy' and the value 'true'",
				CommandLine: "kubectl label pods foo unhealthy=true",
			},
			{
				Description: "Update pod 'foo' with the label 'status' and the value 'unhealthy', overwriting any existing value",
				CommandLine: "kubectl label --overwrite pods foo status=unhealthy",
			},
			{
				Description: "Update all pods in the namespace",
				CommandLine: "kubectl label pods --all status=unhealthy",
			},
			{
				Description: "Update a pod identified by the type and name in \"pod.json\"",
				CommandLine: "kubectl label -f pod.json status=unhealthy",
			},
			{
				Description: "Update pod 'foo' only if the resource is unchanged from version 1",
				CommandLine: "kubectl label pods foo status=unhealthy --resource-version=1",
			},
			{
				Description: "Update pod 'foo' by removing a label named 'bar' if it exists\nDoes not require the --overwrite flag",
				CommandLine: "kubectl label pods foo bar-",
			},
		},
		Declare: func(cb *cli.CommandBuilder[labelConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[labelConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *labelConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag = allFlag.Description("Select all resources, in the namespace of the specified resource types")
			allFlag.Long("all").ImplicitValue(true)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag = allNamespacesFlag.Description("If true, check the specified action in all namespaces.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-label")
			fieldManagerFlag.Long("field-manager")

			fieldSelectorFlag := cb.Flags.String(&cb.Config.FieldSelector)
			fieldSelectorFlag = fieldSelectorFlag.Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files identifying the resource to update the labels")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			listFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.List))
			listFlag = listFlag.Description("If true, display the labels for a given resource.")
			listFlag.Long("list").ImplicitValue(true)

			localFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Local))
			localFlag = localFlag.Description("If true, label will NOT contact api-server but run locally.")
			localFlag.Long("local").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			overwriteFlag = overwriteFlag.Description("If true, allow labels to be overwritten, otherwise reject label updates that overwrite existing labels.")
			overwriteFlag.Long("overwrite").ImplicitValue(true)

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag = recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag = recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			resourceVersionFlag := cb.Flags.String(&cb.Config.ResourceVersion)
			resourceVersionFlag = resourceVersionFlag.Description("If non-empty, the labels update will only succeed if this is the current resource-version for the object. Only valid when specifying a single resource.")
			resourceVersionFlag.Long("resource-version")

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")
		},
		Run: runLabel,
	}
	return cmd
}

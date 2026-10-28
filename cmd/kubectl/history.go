package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutHistoryConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                []string
	AllowMissingTemplateKeys bool
	Filenames                []string
	Kustomize                string
	Output                   string
	Recursive                bool
	Revision                 int64
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func rolloutHistoryCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutHistoryConfig] {
	cmd := cli.Command[rolloutHistoryConfig]{
		Name: "history",
		Description: cli.Description{
			Short: "View rollout history",
			Long:  "View previous rollout revisions and configurations.",
		},
		Examples: []cli.Example{
			{
				Description: "View the rollout history of a deployment",
				CommandLine: "kubectl rollout history deployment/abc",
			},
			{
				Description: "View the details of daemonset revision 3",
				CommandLine: "kubectl rollout history daemonset/abc --revision=3",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutHistoryConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[rolloutHistoryConfig]) {
				form.Sequence(resources, "RESOURCE", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *rolloutHistoryConfig) (clientConfig, []string) { return cfg.Client, cfg.Resources },
						[]string{"deployment", "daemonset", "statefulset"},
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allowMissingTemplateKeys := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissingTemplateKeys, true)
			cb.Flags.ValueFor(allowMissingTemplateKeys).Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)

			filenames := cb.Values.Sequence(&cb.Config.Filenames, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenames).DelimitedBy(",").Description("Filename, directory, or URL to files identifying the resource to get from a server.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize).Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output).Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive)).Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			revisionFlag := cb.Flags.Value(&cb.Config.Revision, int64Language()).Description("See the details, including podTemplate of the revision specified")
			revisionFlag.Long("revision")

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields)).Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)

			cb.Flags.String(&cb.Config.Template).Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").Long("template")

			cb.Rule.AtLeastOne(resources, filenames).And(kustomizeFlag.Destination())
			cb.Rule.AtLeast(revisionFlag.Destination(), int64(0))
		},
		Run: runRolloutHistory,
	}
	return cmd
}

package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutUndoConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Resources                []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	Filenames                []string
	Kustomize                string
	Output                   string
	Recursive                bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	ToRevision               int64
}

func rolloutUndoCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutUndoConfig] {
	cmd := cli.Command[rolloutUndoConfig]{
		Name: "undo",
		Description: cli.Description{
			Short: "Undo a previous rollout",
			Long:  "Roll back to a previous rollout.",
		},
		Examples: []cli.Example{
			{
				Description: "Roll back to the previous deployment",
				CommandLine: "kubectl rollout undo deployment/abc",
			},
			{
				Description: "Roll back to daemonset revision 3",
				CommandLine: "kubectl rollout undo daemonset/abc --to-revision=3",
			},
			{
				Description: "Roll back to the previous deployment with dry-run",
				CommandLine: "kubectl rollout undo --dry-run=server deployment/abc",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutUndoConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[rolloutUndoConfig]) {
				form.Sequence(resources, "RESOURCE", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *rolloutUndoConfig) (clientConfig, []string) { return cfg.Client, cfg.Resources },
						[]string{"deployment", "daemonset", "statefulset"},
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allowMissingTemplateKeys := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissingTemplateKeys, true)
			cb.Flags.ValueFor(allowMissingTemplateKeys).Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)

			dryRun := cb.Values.String(&cb.Config.DryRun)
			cb.Values.Default(dryRun, "none")
			cb.Flags.ValueFor(dryRun).Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.").Long("dry-run").ImplicitValue("unchanged")

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

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields)).Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)

			cb.Flags.String(&cb.Config.Template).Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").Long("template")

			toRevisionFlag := cb.Flags.Value(&cb.Config.ToRevision, int64Language()).Description("The revision to rollback to. Default to 0 (last revision).")
			toRevisionFlag.Long("to-revision")

			cb.Rule.AtLeastOne(resources, filenames).And(kustomizeFlag.Destination())
			cb.Rule.AtLeast(toRevisionFlag.Destination(), int64(0))
		},
		Run: runRolloutUndo,
	}
	return cmd
}

package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type applySetLastAppliedConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	AllowMissingTemplateKeys bool
	CreateAnnotation         bool
	DryRun                   util.DryRunStrategy
	Filename                 []string
	Output                   string
	ShowManagedFields        bool
	Template                 string
}

func applySetLastAppliedCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[applySetLastAppliedConfig] {
	cmd := cli.Command[applySetLastAppliedConfig]{
		Name: "set-last-applied",
		Description: cli.Description{
			Short: "Set the last-applied-configuration annotation on a live object to match the contents of a file",
			Long:  "Set the latest last-applied-configuration annotations by setting it to match the contents of a file. This results in the last-applied-configuration being updated as though 'kubectl apply -f<file> ' was run, without updating any other parts of the object.",
		},
		Examples: []cli.Example{
			{
				Description: "Set the last-applied-configuration of a resource to match the contents of a file",
				CommandLine: "kubectl apply set-last-applied -f deploy.yaml",
			},
			{
				Description: "Execute set-last-applied against each configuration file in a directory",
				CommandLine: "kubectl apply set-last-applied -f path/",
			},
			{
				Description: "Set the last-applied-configuration of a resource to match the contents of a file; will create the annotation if it does not already exist",
				CommandLine: "kubectl apply set-last-applied -f deploy.yaml --create-annotation=true",
			},
		},
		Declare: func(cb *cli.CommandBuilder[applySetLastAppliedConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)
			cb.Values.Default(allowMissingTemplateKeysFlag, true)

			createAnnotationFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.CreateAnnotation))
			createAnnotationFlag.Description("Will create 'last-applied-configuration' annotations if current objects doesn't have one").Long("create-annotation").ImplicitValue(true)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.").Long("dry-run").ImplicitValue(util.DryRunClient)
			cb.Values.Default(dryRun, util.DryRunNone)

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",").Description("Filename, directory, or URL to files that contains the last-applied-configuration annotations").Long("filename")
			filenameFlag.Short('f')

			cb.Rule.Require(filenameValue)
			cb.Rule.ItemCount(filenameValue, cli.Count.OneOrMore())

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).").Long("output")
			outputFlag.Short('o')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").Long("template")
		},
		Run: runApplySetLastApplied,
	}
	return cmd
}

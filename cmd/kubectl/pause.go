package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutPauseConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	AllowMissingTemplateKeys bool
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Output                   string
	Recursive                bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
}

func rolloutPauseCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutPauseConfig] {
	cmd := cli.Command[rolloutPauseConfig]{
		Name: "pause",
		Description: cli.Description{
			Short: "Mark the provided resource as paused",
			Long:  "Mark the provided resource as paused.\n\n Paused resources will not be reconciled by a controller. Use \"kubectl rollout resume\" to resume a paused resource. Currently only deployments support being paused.",
		},
		Examples: []cli.Example{
			{
				Description: "Mark the nginx deployment as paused\nAny current state of the deployment will continue its function; new updates\nto the deployment will not have an effect as long as the deployment is paused",
				CommandLine: "kubectl rollout pause deployment/nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutPauseConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[rolloutPauseConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *rolloutPauseConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"deployment"},
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-rollout")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files identifying the resource to get from a server.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

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
		Run: runRolloutPause,
	}
	return cmd
}

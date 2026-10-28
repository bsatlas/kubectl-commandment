package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type taintConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	All                      bool
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Output                   string
	Overwrite                bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func taintCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[taintConfig] {
	cmd := cli.Command[taintConfig]{
		Name: "taint",
		Description: cli.Description{
			Short: "Update the taints on one or more nodes",
			Long:  "Update the taints on one or more nodes.\n\n  *  A taint consists of a key, value, and effect. As an argument here, it is expressed as key=value:effect.\n  *  The key must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores, up to 253 characters.\n  *  Optionally, the key can begin with a DNS subdomain prefix and a single '/', like example.com/my-app.\n  *  The value is optional. If given, it must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores, up to 63 characters.\n  *  The effect must be NoSchedule, PreferNoSchedule or NoExecute.\n  *  Currently taint can only apply to node.",
		},
		Examples: []cli.Example{
			{
				Description: "Update node 'foo' with a taint with key 'dedicated' and value 'special-user' and effect 'NoSchedule'\nIf a taint with that key and effect already exists, its value is replaced as specified",
				CommandLine: "kubectl taint nodes foo dedicated=special-user:NoSchedule",
			},
			{
				Description: "Remove from node 'foo' the taint with key 'dedicated' and effect 'NoSchedule' if one exists",
				CommandLine: "kubectl taint nodes foo dedicated:NoSchedule-",
			},
			{
				Description: "Remove from node 'foo' all the taints with key 'dedicated'",
				CommandLine: "kubectl taint nodes foo dedicated-",
			},
			{
				Description: "Add a taint with key 'dedicated' on nodes having label myLabel=X",
				CommandLine: "kubectl taint node -l myLabel=X  dedicated=foo:PreferNoSchedule",
			},
			{
				Description: "Add to node 'foo' a taint with key 'bar' and no value",
				CommandLine: "kubectl taint nodes foo bar:NoSchedule",
			},
		},
		Declare: func(cb *cli.CommandBuilder[taintConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[taintConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *taintConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"node"},
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag = allFlag.Description("Select all nodes in the cluster")
			allFlag.Long("all").ImplicitValue(true)

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
			cb.Values.Default(fieldManagerFlag, "kubectl-taint")
			fieldManagerFlag.Long("field-manager")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
			overwriteFlag = overwriteFlag.Description("If true, allow taints to be overwritten, otherwise reject taint updates that overwrite existing taints.")
			overwriteFlag.Long("overwrite").ImplicitValue(true)

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

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runTaint,
	}
	return cmd
}

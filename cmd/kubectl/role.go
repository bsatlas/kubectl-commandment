package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createRoleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Output                   string
	Resources                []string
	ResourceNames            []string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
	Verbs                    []string
}

func createRoleCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createRoleConfig] {
	cmd := cli.Command[createRoleConfig]{
		Name: "role",
		Description: cli.Description{
			Short: "Create a role with single rule",
			Long:  "Create a role with single rule.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a role named \"pod-reader\" that allows user to perform \"get\", \"watch\" and \"list\" on pods",
				CommandLine: "kubectl create role pod-reader --verb=get --verb=list --verb=watch --resource=pods",
			},
			{
				Description: "Create a role named \"pod-reader\" with ResourceName specified",
				CommandLine: "kubectl create role pod-reader --verb=get --resource=pods --resource-name=readablepod --resource-name=anotherpod",
			},
			{
				Description: "Create a role named \"foo\" with API Group specified",
				CommandLine: "kubectl create role foo --verb=get,list,watch --resource=rs.apps",
			},
			{
				Description: "Create a role named \"foo\" with SubResource specified",
				CommandLine: "kubectl create role foo --verb=get,list,watch --resource=pods,pods/status",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createRoleConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createRoleConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createRoleConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

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
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			resourceValue := cb.Values.Sequence(&cb.Config.Resources, cli.Value.String())
			resourceFlag := cb.Flags.Sequence(resourceValue)
			resourceFlag = resourceFlag.DelimitedBy(",")
			resourceFlag = resourceFlag.Description("Resource that the rule applies to")
			resourceFlag.Long("resource")

			resourceNameValue := cb.Values.Sequence(&cb.Config.ResourceNames, cli.Value.String())
			resourceNameFlag := cb.Flags.Sequence(resourceNameValue)
			resourceNameFlag = resourceNameFlag.Description("Resource in the white list that the rule applies to, repeat this flag for multiple items")
			resourceNameFlag.Long("resource-name")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

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

			verbValue := cb.Values.Sequence(&cb.Config.Verbs, cli.Value.String())
			verbFlag := cb.Flags.Sequence(verbValue)
			verbFlag = verbFlag.DelimitedBy(",")
			verbFlag = verbFlag.Description("Verb that applies to the resources contained in the rule")
			verbFlag.Long("verb")
		},
		Run: runCreateRole,
	}
	return cmd
}

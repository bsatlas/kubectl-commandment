package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createClusterRoleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	AggregationRule          map[string]string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	NonResourceURLs          []string
	Output                   string
	Resources                []string
	ResourceNames            []string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
	Verbs                    []string
}

func createClusterRoleCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createClusterRoleConfig] {
	cmd := cli.Command[createClusterRoleConfig]{
		Name: "clusterrole",
		Description: cli.Description{
			Short: "Create a cluster role",
			Long:  "Create a cluster role.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a cluster role named \"pod-reader\" that allows user to perform \"get\", \"watch\" and \"list\" on pods",
				CommandLine: "kubectl create clusterrole pod-reader --verb=get,list,watch --resource=pods",
			},
			{
				Description: "Create a cluster role named \"pod-reader\" with ResourceName specified",
				CommandLine: "kubectl create clusterrole pod-reader --verb=get --resource=pods --resource-name=readablepod --resource-name=anotherpod",
			},
			{
				Description: "Create a cluster role named \"foo\" with API Group specified",
				CommandLine: "kubectl create clusterrole foo --verb=get,list,watch --resource=rs.apps",
			},
			{
				Description: "Create a cluster role named \"foo\" with SubResource specified",
				CommandLine: "kubectl create clusterrole foo --verb=get,list,watch --resource=pods,pods/status",
			},
			{
				Description: "Create a cluster role name \"foo\" with NonResourceURL specified",
				CommandLine: "kubectl create clusterrole \"foo\" --verb=get --non-resource-url=/logs/*",
			},
			{
				Description: "Create a cluster role name \"monitoring\" with AggregationRule specified",
				CommandLine: "kubectl create clusterrole monitoring --aggregation-rule=\"rbac.example.com/aggregate-to-monitoring=true\"",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createClusterRoleConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createClusterRoleConfig]) {
				form.ValueFor(name, "NAME")
			})

			aggregationRuleValue := cb.Values.Keyed(&cb.Config.AggregationRule, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			aggregationRuleFlag := cb.Flags.Keyed(aggregationRuleValue)
			aggregationRuleFlag = aggregationRuleFlag.Description("An aggregation label selector for combining ClusterRoles.")
			aggregationRuleFlag.Long("aggregation-rule")

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

			nonResourceURLValue := cb.Values.Sequence(&cb.Config.NonResourceURLs, cli.Value.String())
			nonResourceURLFlag := cb.Flags.Sequence(nonResourceURLValue)
			nonResourceURLFlag = nonResourceURLFlag.DelimitedBy(",")
			nonResourceURLFlag = nonResourceURLFlag.Description("A partial url that user should have access to.")
			nonResourceURLFlag.Long("non-resource-url")

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
		Run: runCreateClusterRole,
	}
	return cmd
}

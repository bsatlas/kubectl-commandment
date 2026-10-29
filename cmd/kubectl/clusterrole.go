package main

import (
	"context"

	"strings"

	"github.com/abstractionsinc/commandment/cli"

	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
)

func clusterRoleNonResourceURLLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "non-resource-url",
		Parse: func(value string) (string, error) {
			if value == "*" {
				return value, nil
			}
			if value == "" || !strings.HasPrefix(value, "/") {
				return "", cli.Rejection.Message("non-resource-url", "non-resource URL must start with /", cli.Rejection.At(0, 0, len(value)))
			}
			if strings.ContainsRune(value[:len(value)-1], '*') {
				return "", cli.Rejection.Message("non-resource-url", "wildcard is supported only at the end", cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

func validClusterRoleNonResourceVerb(verb string) bool {
	switch verb {
	case "*", "get", "post", "put", "delete", "patch", "head", "options":
		return true
	default:
		return false
	}
}

type createClusterRoleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AggregationRule          map[string]string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	NonResourceURLInputs     []string
	NonResourceURLs          []string
	Output                   string
	ResourceInputs           []create.ResourceOptions
	Resources                []create.ResourceOptions
	ResourceNameInputs       []string
	ResourceNames            []string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
	VerbInputs               []string
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

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createClusterRoleConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createClusterRoleConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			aggregationRuleValue := cb.Values.Keyed(&cb.Config.AggregationRule, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			aggregationRuleFlag := cb.Flags.Keyed(aggregationRuleValue)
			aggregationRuleFlag = aggregationRuleFlag.Description("An aggregation label selector for combining ClusterRoles.")
			aggregationRuleFlag.Long("aggregation-rule")

			cb.Rule.NotEmpty(name)

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

			nonResourceURLInput := cb.Values.Sequence(&cb.Config.NonResourceURLInputs, clusterRoleNonResourceURLLanguage())
			nonResourceURLs := cb.Values.Derive(&cb.Config.NonResourceURLs, cli.Value.ComputedBy(func(_ context.Context, _ *createClusterRoleConfig, dependencies cli.Dependencies) ([]string, error) {
				values, _ := dependencies.Get(nonResourceURLInput)
				return uniqueCreateStrings(values), nil
			}).From(nonResourceURLInput.OptionalDependency()))
			nonResourceURLFlag := cb.Flags.Sequence(nonResourceURLInput)
			nonResourceURLFlag = nonResourceURLFlag.DelimitedBy(",")
			nonResourceURLFlag = nonResourceURLFlag.Description("A partial url that user should have access to.")
			nonResourceURLFlag.Long("non-resource-url")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			resourceInput := cb.Values.Sequence(&cb.Config.ResourceInputs, createRoleResourceLanguage())
			resources := cb.Values.Derive(&cb.Config.Resources, cli.Value.ComputedBy(func(_ context.Context, _ *createClusterRoleConfig, dependencies cli.Dependencies) ([]create.ResourceOptions, error) {
				values, _ := dependencies.Get(resourceInput)
				return normalizeCreateRoleResources(values), nil
			}).From(resourceInput.OptionalDependency()))
			resourceFlag := cb.Flags.Sequence(resourceInput)
			resourceFlag = resourceFlag.DelimitedBy(",")
			resourceFlag = resourceFlag.Description("Resource that the rule applies to")
			resourceFlag.Long("resource")

			resourceNameInput := cb.Values.Sequence(&cb.Config.ResourceNameInputs, cli.Value.String())
			resourceNames := cb.Values.Derive(&cb.Config.ResourceNames, cli.Value.ComputedBy(func(_ context.Context, _ *createClusterRoleConfig, dependencies cli.Dependencies) ([]string, error) {
				values, _ := dependencies.Get(resourceNameInput)
				return uniqueCreateStrings(values), nil
			}).From(resourceNameInput.OptionalDependency()))
			resourceNameFlag := cb.Flags.Sequence(resourceNameInput)
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

			verbInput := cb.Values.Sequence(&cb.Config.VerbInputs, cli.Value.String())
			verbs := cb.Values.Derive(&cb.Config.Verbs, cli.Value.ComputedBy(func(_ context.Context, _ *createClusterRoleConfig, dependencies cli.Dependencies) ([]string, error) {
				values, _ := dependencies.Get(verbInput)
				return uniqueCreateWildcardStrings(values), nil
			}).From(verbInput.OptionalDependency()))
			verbFlag := cb.Flags.Sequence(verbInput)
			verbFlag = verbFlag.DelimitedBy(",")
			verbFlag = verbFlag.Description("Verb that applies to the resources contained in the rule")
			verbFlag.Long("verb")

			cb.Rule.Check("kubectl-create-clusterrole-shape", func(_ context.Context, cfg *createClusterRoleConfig, _ cli.Dependencies) (bool, error) {
				if len(cfg.AggregationRule) > 0 {
					return len(cfg.NonResourceURLs) == 0 && len(cfg.Verbs) == 0 && len(cfg.Resources) == 0 && len(cfg.ResourceNames) == 0, nil
				}
				return len(cfg.Verbs) > 0 && (len(cfg.Resources) > 0 || len(cfg.NonResourceURLs) > 0), nil
			}).
				DependsOn(aggregationRuleValue.OptionalDependency()).
				DependsOn(nonResourceURLs).
				DependsOn(verbs).
				DependsOn(resources).
				DependsOn(resourceNames)
			cb.Rule.Check("kubectl-create-clusterrole-non-resource-verbs", func(_ context.Context, cfg *createClusterRoleConfig, _ cli.Dependencies) (bool, error) {
				if len(cfg.NonResourceURLs) == 0 {
					return true, nil
				}
				for _, verb := range cfg.Verbs {
					if !validClusterRoleNonResourceVerb(verb) {
						return false, nil
					}
				}
				return true, nil
			}).
				DependsOn(nonResourceURLs).
				DependsOn(verbs)
		},
		Run: runCreateClusterRole,
	}
	return cmd
}

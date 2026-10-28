package kubectl

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	"k8s.io/kubectl/pkg/util"
	"k8s.io/utils/net"
)

type createConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	AllowMissingTemplateKeys bool
	Direct                   bool
	DryRun                   string
	Edit                     bool
	FieldManager             string
	Filename                 []string
	Kustomize                string
	Output                   string
	Raw                      string
	Record                   bool
	Recursive                bool
	SaveConfig               bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
	WindowsLineEndings       bool
}

func createCommand(client cli.Destination[rootConfig, clientConfig], plugins []pluginNode) cli.Command[createConfig] {
	cmd := cli.Command[createConfig]{
		Name: "create",
		Description: cli.Description{
			Short: "Create a resource from a file or from stdin",
			Long:  "Create a resource from a file or from stdin.\n\n JSON and YAML formats are accepted.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a pod using the data in pod.json",
				CommandLine: "kubectl create -f ./pod.json",
			},
			{
				Description: "Create a pod based on the JSON passed into stdin",
				CommandLine: "cat pod.json | kubectl create -f -",
			},
			{
				Description: "Edit the data in registry.yaml in JSON then create the resource using the edited data",
				CommandLine: "kubectl create -f registry.yaml --edit -o json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			direct := cb.Values.Bool(&cb.Config.Direct)
			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[createConfig]) {}).Selects(direct, true)
			directRules := cb.Rule.When(cb.Rule.ContributedBy(direct, cli.ValueProducerKindArgument))

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			editFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Edit))
			editFlag = editFlag.Description("Edit the API resource before creating")
			editFlag.Long("edit").ImplicitValue(true)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files to use to create the resource")
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			rawFlag := cb.Flags.String(&cb.Config.Raw)
			rawFlag = rawFlag.Description("Raw URI to POST to the server.  Uses the transport specified by the kubeconfig file.")
			rawFlag.Long("raw")

			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			recordFlag = recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag = recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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

			windowsLineEndingsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.WindowsLineEndings))
			windowsLineEndingsFlag = windowsLineEndingsFlag.Description("Only relevant if --edit=true. Defaults to the line ending native to your platform.")
			windowsLineEndingsFlag.Long("windows-line-endings").ImplicitValue(true)

			directRules.Check("kubectl-create-source", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				filename := len(cfg.Filename) > 0
				kustomize := cfg.Kustomize != ""
				return filename != kustomize, nil
			}).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomizeFlag.Destination().OptionalDependency())
			directRules.Check("kubectl-create-kustomize-recursive", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Kustomize == "" || !cfg.Recursive, nil
			}).
				DependsOn(kustomizeFlag.Destination().OptionalDependency()).
				DependsOn(recursiveFlag.Destination().OptionalDependency())
			directRules.Check("kubectl-create-raw-file", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" {
					return true, nil
				}
				if len(cfg.Filename) != 1 {
					return false, nil
				}
				filename := cfg.Filename[0]
				return !strings.HasPrefix(filename, "http://") && !strings.HasPrefix(filename, "https://"), nil
			}).
				DependsOn(rawFlag.Destination().OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			directRules.Check("kubectl-create-raw-edit", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw == "" || !cfg.Edit, nil
			}).
				DependsOn(rawFlag.Destination().OptionalDependency()).
				DependsOn(editFlag.Destination().OptionalDependency())
			directRules.Check("kubectl-create-raw-recursive", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw == "" || !cfg.Recursive, nil
			}).
				DependsOn(rawFlag.Destination().OptionalDependency()).
				DependsOn(recursiveFlag.Destination().OptionalDependency())
			directRules.Check("kubectl-create-raw-selector", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw == "" || cfg.Selector == "", nil
			}).
				DependsOn(rawFlag.Destination().OptionalDependency()).
				DependsOn(selectorFlag.Destination().OptionalDependency())
			directRules.Check("kubectl-create-raw-output", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw == "" || cfg.Output == "", nil
			}).
				DependsOn(rawFlag.Destination().OptionalDependency()).
				DependsOn(outputFlag.Destination().OptionalDependency())
			directRules.Check("kubectl-create-raw-uri", func(_ context.Context, cfg *createConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" {
					return true, nil
				}
				_, err := url.ParseRequestURI(cfg.Raw)
				return err == nil, nil
			}).DependsOn(rawFlag.Destination().OptionalDependency())

			clusterRole := createClusterRoleCommand(client)
			cb.Subcommands.Command(clusterRole)
			clusterRoleBinding := createClusterRoleBindingCommand(client)
			cb.Subcommands.Command(clusterRoleBinding)
			configMap := createConfigMapCommand(client)
			cb.Subcommands.Command(configMap)
			cronJob := createCronJobCommand(client)
			cb.Subcommands.Command(cronJob)
			deployment := createDeploymentCommand(client)
			cb.Subcommands.Command(deployment)
			ingress := createIngressCommand(client)
			cb.Subcommands.Command(ingress)
			job := createJobCommand(client)
			cb.Subcommands.Command(job)
			namespace := createNamespaceCommand(client)
			cb.Subcommands.Command(namespace)
			podDisruptionBudget := createPodDisruptionBudgetCommand(client)
			cb.Subcommands.Command(podDisruptionBudget)
			priorityClass := createPriorityClassCommand(client)
			cb.Subcommands.Command(priorityClass)
			quota := createQuotaCommand(client)
			cb.Subcommands.Command(quota)
			role := createRoleCommand(client)
			cb.Subcommands.Command(role)
			roleBinding := createRoleBindingCommand(client)
			cb.Subcommands.Command(roleBinding)
			secret := createSecretCommand(client)
			cb.Subcommands.Command(secret)
			service := createServiceCommand(client)
			cb.Subcommands.Command(service)
			serviceAccount := createServiceAccountCommand(client)
			cb.Subcommands.Command(serviceAccount)
			token := createTokenCommand(client)
			cb.Subcommands.Command(token)

			occupied := pluginOccupiedNames(
				clusterRole.Name, clusterRoleBinding.Name, configMap.Name, cronJob.Name, deployment.Name, ingress.Name,
				job.Name, namespace.Name, podDisruptionBudget.Name, priorityClass.Name, quota.Name, role.Name, roleBinding.Name,
				secret.Name, service.Name, serviceAccount.Name, token.Name,
			)
			addPluginProviders(&cb.Subcommands, plugins, occupied)
		},
		Run: runCreate,
	}
	return cmd
}

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

type createClusterRoleBindingConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	ClusterRole              string
	DryRun                   string
	FieldManager             string
	Groups                   []string
	Output                   string
	SaveConfig               bool
	ServiceAccounts          []string
	ShowManagedFields        bool
	Template                 string
	Users                    []string
	Validate                 string
}

func createClusterRoleBindingCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createClusterRoleBindingConfig] {
	cmd := cli.Command[createClusterRoleBindingConfig]{
		Name: "clusterrolebinding",
		Description: cli.Description{
			Short: "Create a cluster role binding for a particular cluster role",
			Long:  "Create a cluster role binding for a particular cluster role.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a cluster role binding for user1, user2, and group1 using the cluster-admin cluster role",
				CommandLine: "kubectl create clusterrolebinding cluster-admin --clusterrole=cluster-admin --user=user1 --user=user2 --group=group1",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createClusterRoleBindingConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createClusterRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createClusterRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			clusterRoleFlag := cb.Flags.String(&cb.Config.ClusterRole)
			clusterRoleFlag = clusterRoleFlag.Description("ClusterRole this ClusterRoleBinding should reference")
			clusterRoleFlag.CompleteBy(fixedResourceArgumentCompleter(
				func(cfg *createClusterRoleBindingConfig) (clientConfig, []string) { return cfg.Client, nil },
				"clusterrole",
			)).From(client)
			clusterRoleFlag.Long("clusterrole")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			groupValue := cb.Values.Sequence(&cb.Config.Groups, cli.Value.String())
			groupFlag := cb.Flags.Sequence(groupValue)
			groupFlag = groupFlag.Description("Groups to bind to the clusterrole. The flag can be repeated to add multiple groups.")
			groupFlag.Long("group")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			serviceAccountValue := cb.Values.Sequence(&cb.Config.ServiceAccounts, roleBindingServiceAccountLanguage())
			serviceAccountFlag := cb.Flags.Sequence(serviceAccountValue)
			serviceAccountFlag = serviceAccountFlag.Description("Service accounts to bind to the clusterrole, in the format <namespace>:<name>. The flag can be repeated to add multiple service accounts.")
			serviceAccountFlag.Long("serviceaccount")

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			userValue := cb.Values.Sequence(&cb.Config.Users, cli.Value.String())
			userFlag := cb.Flags.Sequence(userValue)
			userFlag = userFlag.Description("Usernames to bind to the clusterrole. The flag can be repeated to add multiple users.")
			userFlag.Long("user")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Require(clusterRoleFlag.Destination())
		},
		Run: runCreateClusterRoleBinding,
	}
	return cmd
}

func createFileSourceLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "file-source",
		Parse: func(value string) (string, error) {
			if _, _, err := util.ParseFileSource(value); err != nil {
				return "", cli.Rejection.Message("file-source", err.Error(), cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

func createLiteralSourceLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "literal-source",
		Parse: func(value string) (string, error) {
			if _, _, err := util.ParseLiteralSource(value); err != nil {
				return "", cli.Rejection.Message("literal-source", err.Error(), cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createConfigMapConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	DryRun                   string
	FieldManager             string
	FromEnvFile              []string
	FromFile                 []string
	FromLiteral              []string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createConfigMapCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createConfigMapConfig] {
	cmd := cli.Command[createConfigMapConfig]{
		Name:    "configmap",
		Aliases: cli.Aliases("cm"),
		Description: cli.Description{
			Short: "Create a config map from a local file, directory or literal value",
			Long:  "Create a config map based on a file, directory, or specified literal value.\n\n A single config map may package one or more key/value pairs.\n\n When creating a config map based on a file, the key will default to the basename of the file, and the value will default to the file content.  If the basename is an invalid key, you may specify an alternate key.\n\n When creating a config map based on a directory, each file whose basename is a valid key in the directory will be packaged into the config map.  Any directory entries except regular files are ignored (e.g. subdirectories, symlinks, devices, pipes, etc).",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new config map named my-config based on folder bar",
				CommandLine: "kubectl create configmap my-config --from-file=path/to/bar",
			},
			{
				Description: "Create a new config map named my-config with specified keys instead of file basenames on disk",
				CommandLine: "kubectl create configmap my-config --from-file=key1=/path/to/bar/file1.txt --from-file=key2=/path/to/bar/file2.txt",
			},
			{
				Description: "Create a new config map named my-config with key1=config1 and key2=config2",
				CommandLine: "kubectl create configmap my-config --from-literal=key1=config1 --from-literal=key2=config2",
			},
			{
				Description: "Create a new config map named my-config with key=value pairs from an env file",
				CommandLine: "kubectl create configmap my-config --from-env-file=path/to/bar.env",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createConfigMapConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createConfigMapConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createConfigMapConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			cb.Rule.NotEmpty(name)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			appendHashFlag = appendHashFlag.Description("Append a hash of the configmap to its name.")
			appendHashFlag.Long("append-hash").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			fromEnvFileValue := cb.Values.Sequence(&cb.Config.FromEnvFile, cli.Value.String())
			fromEnvFileFlag := cb.Flags.Sequence(fromEnvFileValue)
			fromEnvFileFlag = fromEnvFileFlag.DelimitedBy(",")
			fromEnvFileFlag = fromEnvFileFlag.Description("Specify the path to a file to read lines of key=val pairs to create a configmap.")
			fromEnvFileFlag.Long("from-env-file")

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, createFileSourceLanguage())
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag = fromFileFlag.DelimitedBy(",")
			fromFileFlag = fromFileFlag.Description("Key file can be specified using its file path, in which case file basename will be used as configmap key, or optionally with a key and file path, in which case the given key will be used.  Specifying a directory will iterate each named file in the directory whose basename is a valid configmap key.")
			fromFileFlag.Long("from-file")

			fromLiteralValue := cb.Values.Sequence(&cb.Config.FromLiteral, createLiteralSourceLanguage())
			fromLiteralFlag := cb.Flags.Sequence(fromLiteralValue)
			fromLiteralFlag = fromLiteralFlag.Description("Specify a key and literal value to insert in configmap (i.e. mykey=somevalue)")
			fromLiteralFlag.Long("from-literal")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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

			cb.Rule.When(cb.When.Resolved(fromEnvFileValue)).Forbid(fromFileValue)
			cb.Rule.When(cb.When.Resolved(fromEnvFileValue)).Forbid(fromLiteralValue)
		},
		Run: runCreateConfigMap,
	}
	return cmd
}

type createCronJobConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Command                  []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Image                    string
	Output                   string
	Restart                  string
	SaveConfig               bool
	Schedule                 string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createCronJobCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createCronJobConfig] {
	cmd := cli.Command[createCronJobConfig]{
		Name:    "cronjob",
		Aliases: cli.Aliases("cj"),
		Description: cli.Description{
			Short: "Create a cron job with the specified name",
			Long:  "Create a cron job with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a cron job",
				CommandLine: "kubectl create cronjob my-job --image=busybox --schedule=\"*/1 * * * *\"",
			},
			{
				Description: "Create a cron job with a command",
				CommandLine: "kubectl create cronjob my-job --image=busybox --schedule=\"*/1 * * * *\" -- date",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createCronJobConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			command := cb.Values.Sequence(&cb.Config.Command, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createCronJobConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createCronJobConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(command, "COMMAND", cli.Count.ZeroOrMore())
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

			imageFlag := cb.Flags.String(&cb.Config.Image)
			imageFlag = imageFlag.Description("Image name to run.")
			imageFlag.Long("image")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			restartFlag := cb.Flags.String(&cb.Config.Restart)
			restartFlag = restartFlag.Description("job's restart policy. supported values: OnFailure, Never")
			cb.Values.Default(restartFlag, "OnFailure")
			restartFlag.Long("restart")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			scheduleFlag := cb.Flags.String(&cb.Config.Schedule)
			scheduleFlag = scheduleFlag.Description("A schedule in the Cron format the job should be run with.")
			scheduleFlag.Long("schedule")

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

			cb.Rule.Require(imageFlag.Destination())
			cb.Rule.Require(scheduleFlag.Destination())
		},
		Run: runCreateCronJob,
	}
	return cmd
}

type createDeploymentConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Command                  []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Images                   []string
	Output                   string
	Port                     int32
	Replicas                 int32
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createDeploymentCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createDeploymentConfig] {
	cmd := cli.Command[createDeploymentConfig]{
		Name:    "deployment",
		Aliases: cli.Aliases("deploy"),
		Description: cli.Description{
			Short: "Create a deployment with the specified name",
			Long:  "Create a deployment with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a deployment named my-dep that runs the busybox image",
				CommandLine: "kubectl create deployment my-dep --image=busybox",
			},
			{
				Description: "Create a deployment with a command",
				CommandLine: "kubectl create deployment my-dep --image=busybox -- date",
			},
			{
				Description: "Create a deployment named my-dep that runs the nginx image with 3 replicas",
				CommandLine: "kubectl create deployment my-dep --image=nginx --replicas=3",
			},
			{
				Description: "Create a deployment named my-dep that runs the busybox image and expose port 5701",
				CommandLine: "kubectl create deployment my-dep --image=busybox --port=5701",
			},
			{
				Description: "Create a deployment named my-dep that runs multiple containers",
				CommandLine: "kubectl create deployment my-dep --image=busybox:latest --image=ubuntu:latest --image=nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createDeploymentConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			command := cb.Values.Sequence(&cb.Config.Command, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createDeploymentConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createDeploymentConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(command, "COMMAND", cli.Count.ZeroOrMore())
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

			imageValue := cb.Values.Sequence(&cb.Config.Images, cli.Value.String())
			imageFlag := cb.Flags.Sequence(imageValue)
			imageFlag = imageFlag.DelimitedBy(",")
			imageFlag = imageFlag.Description("Image names to run. A deployment can have multiple images set for multi-container pod.")
			imageFlag.Long("image")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			portFlag := cb.Flags.Value(&cb.Config.Port, int32Language())
			portFlag = portFlag.Description("The containerPort that this deployment exposes.")
			cb.Values.Default(portFlag, int32(-1))
			portFlag.Long("port")

			replicasFlag := cb.Flags.Value(&cb.Config.Replicas, int32Language())
			replicasFlag = replicasFlag.Description("Number of replicas to create. Default is 1.")
			cb.Values.Default(replicasFlag, int32(1))
			replicasFlag.Long("replicas")
			replicasFlag.Short('r').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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

			cb.Rule.Require(imageValue)
			cb.Rule.Check("kubectl-create-deployment-images-command", func(_ context.Context, cfg *createDeploymentConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Images) <= 1 || len(cfg.Command) == 0, nil
			}).
				DependsOn(imageValue.OptionalDependency()).
				DependsOn(command.OptionalDependency())
		},
		Run: runCreateDeployment,
	}
	return cmd
}

func ingressRuleLanguage() cli.ValueCodec[string] {
	rule := regexp.MustCompile(`^(?P<host>[\w\*\-\.]*)(?P<path>/[^,=\n]*)=(?P<svcname>[\w\-]+):(?P<svcport>[\w\-]+)(,(?P<istls>tls)=?(?P<secretname>[\w\-]+)?)?$`)
	codec := cli.ValueCodec[string]{
		Name: "ingress-rule",
		Parse: func(value string) (string, error) {
			if !rule.MatchString(value) {
				return "", cli.Rejection.Message("ingress-rule", "expected host/path=service:port[,tls[=secret]]", cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

func ingressAnnotationLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "ingress-annotation",
		Parse: func(value string) (string, error) {
			parts := strings.SplitN(value, "=", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return "", cli.Rejection.Message("ingress-annotation", "expected key=value", cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

func ingressBackendLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "ingress-backend",
		Parse: func(value string) (string, error) {
			parts := strings.Split(value, ":")
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return "", cli.Rejection.Message("ingress-backend", "expected service:port", cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createIngressConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	Annotations              []string
	Class                    string
	DefaultBackend           string
	DryRun                   string
	FieldManager             string
	Output                   string
	Rules                    []string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createIngressCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createIngressConfig] {
	cmd := cli.Command[createIngressConfig]{
		Name:    "ingress",
		Aliases: cli.Aliases("ing"),
		Description: cli.Description{
			Short: "Create an ingress with the specified name",
			Long:  "Create an ingress with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a single ingress called 'simple' that directs requests to foo.com/bar to svc\nsvc1:8080 with a TLS secret \"my-cert\"",
				CommandLine: "kubectl create ingress simple --rule=\"foo.com/bar=svc1:8080,tls=my-cert\"",
			},
			{
				Description: "Create a catch all ingress of \"/path\" pointing to service svc:port and Ingress Class as \"otheringress\"",
				CommandLine: "kubectl create ingress catch-all --class=otheringress --rule=\"/path=svc:port\"",
			},
			{
				Description: "Create an ingress with two annotations: ingress.annotation1 and ingress.annotations2",
				CommandLine: "kubectl create ingress annotated --class=default --rule=\"foo.com/bar=svc:port\" \\\n  --annotation ingress.annotation1=foo \\\n  --annotation ingress.annotation2=bla",
			},
			{
				Description: "Create an ingress with the same host and multiple paths",
				CommandLine: "kubectl create ingress multipath --class=default \\\n  --rule=\"foo.com/=svc:port\" \\\n  --rule=\"foo.com/admin/=svcadmin:portadmin\"",
			},
			{
				Description: "Create an ingress with multiple hosts and the pathType as Prefix",
				CommandLine: "kubectl create ingress ingress1 --class=default \\\n  --rule=\"foo.com/path*=svc:8080\" \\\n  --rule=\"bar.com/admin*=svc2:http\"",
			},
			{
				Description: "Create an ingress with TLS enabled using the default ingress certificate and different path types",
				CommandLine: "kubectl create ingress ingtls --class=default \\\n  --rule=\"foo.com/=svc:https,tls\" \\\n  --rule=\"foo.com/path/subpath*=othersvc:8080\"",
			},
			{
				Description: "Create an ingress with TLS enabled using a specific secret and pathType as Prefix",
				CommandLine: "kubectl create ingress ingsecret --class=default \\\n  --rule=\"foo.com/*=svc:8080,tls=secret1\"",
			},
			{
				Description: "Create an ingress with a default backend",
				CommandLine: "kubectl create ingress ingdefault --class=default \\\n  --default-backend=defaultsvc:http \\\n  --rule=\"foo.com/*=svc:8080,tls=secret1\"",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createIngressConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createIngressConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createIngressConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			annotationValue := cb.Values.Sequence(&cb.Config.Annotations, ingressAnnotationLanguage())
			annotationFlag := cb.Flags.Sequence(annotationValue)
			annotationFlag = annotationFlag.Description("Annotation to insert in the ingress object, in the format annotation=value")
			annotationFlag.Long("annotation")

			classFlag := cb.Flags.String(&cb.Config.Class)
			classFlag = classFlag.Description("Ingress Class to be used")
			classFlag.Long("class")

			defaultBackend := cb.Values.Value(&cb.Config.DefaultBackend, ingressBackendLanguage())
			defaultBackendFlag := cb.Flags.ValueFor(defaultBackend)
			defaultBackendFlag = defaultBackendFlag.Description("Default service for backend, in format of svcname:port")
			defaultBackendFlag.Long("default-backend")

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
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			ruleValue := cb.Values.Sequence(&cb.Config.Rules, ingressRuleLanguage())
			ruleFlag := cb.Flags.Sequence(ruleValue)
			ruleFlag = ruleFlag.Description("Rule in format host/path=service:port[,tls=secretname]. Paths containing the leading character '*' are considered pathType=Prefix. tls argument is optional.")
			ruleFlag.Long("rule")

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

			cb.Rule.Check("kubectl-create-ingress-backend-or-rule", func(_ context.Context, cfg *createIngressConfig, _ cli.Dependencies) (bool, error) {
				return cfg.DefaultBackend != "" || len(cfg.Rules) > 0, nil
			}).
				DependsOn(defaultBackend.OptionalDependency()).
				DependsOn(ruleValue.OptionalDependency())
		},
		Run: runCreateIngress,
	}
	return cmd
}

type createJobConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Command                  []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	From                     string
	Image                    string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createJobCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createJobConfig] {
	cmd := cli.Command[createJobConfig]{
		Name: "job",
		Description: cli.Description{
			Short: "Create a job with the specified name",
			Long:  "Create a job with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a job",
				CommandLine: "kubectl create job my-job --image=busybox",
			},
			{
				Description: "Create a job with a command",
				CommandLine: "kubectl create job my-job --image=busybox -- date",
			},
			{
				Description: "Create a job from a cron job named \"a-cronjob\"",
				CommandLine: "kubectl create job test-job --from=cronjob/a-cronjob",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createJobConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			command := cb.Values.Sequence(&cb.Config.Command, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createJobConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createJobConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(command, "COMMAND", cli.Count.ZeroOrMore())
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

			fromFlag := cb.Flags.String(&cb.Config.From)
			fromFlag = fromFlag.Description("The name of the resource to create a Job from (only CronJob is supported).")
			fromFlag.Long("from")

			imageFlag := cb.Flags.String(&cb.Config.Image)
			imageFlag = imageFlag.Description("Image name to run.")
			imageFlag.Long("image")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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

			cb.Rule.Check("kubectl-create-job-source", func(_ context.Context, cfg *createJobConfig, _ cli.Dependencies) (bool, error) {
				image := cfg.Image != ""
				from := cfg.From != ""
				return image != from, nil
			}).
				DependsOn(imageFlag.Destination().OptionalDependency()).
				DependsOn(fromFlag.Destination().OptionalDependency())
			cb.Rule.Check("kubectl-create-job-from-command", func(_ context.Context, cfg *createJobConfig, _ cli.Dependencies) (bool, error) {
				return cfg.From == "" || len(cfg.Command) == 0, nil
			}).
				DependsOn(fromFlag.Destination().OptionalDependency()).
				DependsOn(command.OptionalDependency())
		},
		Run: runCreateJob,
	}
	return cmd
}

type createNamespaceConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createNamespaceCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createNamespaceConfig] {
	cmd := cli.Command[createNamespaceConfig]{
		Name:    "namespace",
		Aliases: cli.Aliases("ns"),
		Description: cli.Description{
			Short: "Create a namespace with the specified name",
			Long:  "Create a namespace with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new namespace named my-namespace",
				CommandLine: "kubectl create namespace my-namespace",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createNamespaceConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createNamespaceConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createNamespaceConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

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

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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
		},
		Run: runCreateNamespace,
	}
	return cmd
}

func podDisruptionBudgetSelectorLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "label-selector",
		Parse: func(value string) (string, error) {
			if _, err := v1.ParseToLabelSelector(value); err != nil {
				return "", cli.Rejection.Message("label-selector", err.Error(), cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

func podDisruptionBudgetAvailabilityLanguage() cli.ValueCodec[string] {
	valid := regexp.MustCompile(`^[0-9]+%?$`)
	codec := cli.ValueCodec[string]{
		Name: "availability",
		Parse: func(value string) (string, error) {
			if !valid.MatchString(value) {
				return "", cli.Rejection.Message("availability", "expected a non-negative integer or percentage", cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createPodDisruptionBudgetConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	MaxUnavailable           string
	MinAvailable             string
	Output                   string
	SaveConfig               bool
	Selector                 string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createPodDisruptionBudgetCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createPodDisruptionBudgetConfig] {
	cmd := cli.Command[createPodDisruptionBudgetConfig]{
		Name:    "poddisruptionbudget",
		Aliases: cli.Aliases("pdb"),
		Description: cli.Description{
			Short: "Create a pod disruption budget with the specified name",
			Long:  "Create a pod disruption budget with the specified name, selector, and desired minimum available pods.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a pod disruption budget named my-pdb that will select all pods with the app=rails label\nand require at least one of them being available at any point in time",
				CommandLine: "kubectl create poddisruptionbudget my-pdb --selector=app=rails --min-available=1",
			},
			{
				Description: "Create a pod disruption budget named my-pdb that will select all pods with the app=nginx label\nand require at least half of the pods selected to be available at any point in time",
				CommandLine: "kubectl create pdb my-pdb --selector=app=nginx --min-available=50%",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createPodDisruptionBudgetConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createPodDisruptionBudgetConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createPodDisruptionBudgetConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

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

			maxUnavailable := cb.Values.Value(&cb.Config.MaxUnavailable, podDisruptionBudgetAvailabilityLanguage())
			maxUnavailableFlag := cb.Flags.ValueFor(maxUnavailable)
			maxUnavailableFlag = maxUnavailableFlag.Description("The maximum number or percentage of unavailable pods this budget requires.")
			maxUnavailableFlag.Long("max-unavailable")

			minAvailable := cb.Values.Value(&cb.Config.MinAvailable, podDisruptionBudgetAvailabilityLanguage())
			minAvailableFlag := cb.Flags.ValueFor(minAvailable)
			minAvailableFlag = minAvailableFlag.Description("The minimum number or percentage of available pods this budget requires.")
			minAvailableFlag.Long("min-available")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			selectorFlag := cb.Flags.Value(&cb.Config.Selector, podDisruptionBudgetSelectorLanguage())
			selectorFlag = selectorFlag.Description("A label selector to use for this budget. Only equality-based selector requirements are supported.")
			selectorFlag.Long("selector")

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

			cb.Rule.Require(selectorFlag.Destination())
			cb.Rule.NotEmpty(selectorFlag.Destination())
			cb.Rule.ExactlyOne(minAvailable, maxUnavailable)
		},
		Run: runCreatePodDisruptionBudget,
	}
	return cmd
}

type createPriorityClassConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	Description              string
	DryRun                   string
	FieldManager             string
	GlobalDefault            bool
	Output                   string
	PreemptionPolicy         string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
	Value                    int32
}

func createPriorityClassCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createPriorityClassConfig] {
	cmd := cli.Command[createPriorityClassConfig]{
		Name:    "priorityclass",
		Aliases: cli.Aliases("pc"),
		Description: cli.Description{
			Short: "Create a priority class with the specified name",
			Long:  "Create a priority class with the specified name, value, globalDefault and description.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a priority class named high-priority",
				CommandLine: "kubectl create priorityclass high-priority --value=1000 --description=\"high priority\"",
			},
			{
				Description: "Create a priority class named default-priority that is considered as the global default priority",
				CommandLine: "kubectl create priorityclass default-priority --value=1000 --global-default=true --description=\"default priority\"",
			},
			{
				Description: "Create a priority class named high-priority that cannot preempt pods with lower priority",
				CommandLine: "kubectl create priorityclass high-priority --value=1000 --description=\"high priority\" --preemption-policy=\"Never\"",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createPriorityClassConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createPriorityClassConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createPriorityClassConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			descriptionFlag := cb.Flags.String(&cb.Config.Description)
			descriptionFlag = descriptionFlag.Description("description is an arbitrary string that usually provides guidelines on when this priority class should be used.")
			descriptionFlag.Long("description")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			globalDefaultFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.GlobalDefault))
			globalDefaultFlag = globalDefaultFlag.Description("global-default specifies whether this PriorityClass should be considered as the default priority.")
			globalDefaultFlag.Long("global-default").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			preemptionPolicyFlag := cb.Flags.String(&cb.Config.PreemptionPolicy)
			preemptionPolicyFlag = preemptionPolicyFlag.Description("preemption-policy is the policy for preempting pods with lower priority.")
			cb.Values.Default(preemptionPolicyFlag, "PreemptLowerPriority")
			preemptionPolicyFlag.Long("preemption-policy")

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

			valueFlag := cb.Flags.Value(&cb.Config.Value, int32Language())
			valueFlag = valueFlag.Description("the value of this priority class.")
			valueFlag.Long("value")
		},
		Run: runCreatePriorityClass,
	}
	return cmd
}

func quotaHardLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "resource-quota-hard",
		Parse: func(value string) (string, error) {
			if value == "" {
				return value, nil
			}
			for _, statement := range strings.Split(value, ",") {
				parts := strings.Split(statement, "=")
				if len(parts) != 2 {
					return "", cli.Rejection.Message("resource-quota-hard", "expected resource=value", cli.Rejection.At(0, 0, len(value)))
				}
				if _, err := resource.ParseQuantity(parts[1]); err != nil {
					return "", cli.Rejection.Message("resource-quota-hard", err.Error(), cli.Rejection.At(0, 0, len(value)))
				}
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

func quotaScopesLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "resource-quota-scopes",
		Parse: func(value string) (string, error) {
			if value == "" {
				return value, nil
			}
			for _, scope := range strings.Split(value, ",") {
				if scope == "" {
					return "", cli.Rejection.Message("resource-quota-scopes", "scope must not be empty", cli.Rejection.At(0, 0, len(value)))
				}
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createQuotaConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Hard                     string
	Output                   string
	SaveConfig               bool
	Scopes                   string
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createQuotaCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createQuotaConfig] {
	cmd := cli.Command[createQuotaConfig]{
		Name:    "quota",
		Aliases: cli.Aliases("resourcequota"),
		Description: cli.Description{
			Short: "Create a quota with the specified name",
			Long:  "Create a resource quota with the specified name, hard limits, and optional scopes.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new resource quota named my-quota",
				CommandLine: "kubectl create quota my-quota --hard=cpu=1,memory=1G,pods=2,services=3,replicationcontrollers=2,resourcequotas=1,secrets=5,persistentvolumeclaims=10",
			},
			{
				Description: "Create a new resource quota named best-effort",
				CommandLine: "kubectl create quota best-effort --hard=pods=100 --scopes=BestEffort",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createQuotaConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createQuotaConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createQuotaConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

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

			hardFlag := cb.Flags.Value(&cb.Config.Hard, quotaHardLanguage())
			hardFlag = hardFlag.Description("A comma-delimited set of resource=quantity pairs that define a hard limit.")
			hardFlag.Long("hard")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			scopesFlag := cb.Flags.Value(&cb.Config.Scopes, quotaScopesLanguage())
			scopesFlag = scopesFlag.Description("A comma-delimited set of quota scopes that must all match each object tracked by the quota.")
			scopesFlag.Long("scopes")

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
		Run: runCreateQuota,
	}
	return cmd
}

func createRoleResourceLanguage() cli.ValueCodec[create.ResourceOptions] {
	codec := cli.ValueCodec[create.ResourceOptions]{
		Name: "rbac-resource",
		Parse: func(value string) (create.ResourceOptions, error) {
			sections := strings.SplitN(value, "/", 2)
			resource := create.ResourceOptions{}
			if len(sections) == 2 {
				resource.SubResource = sections[1]
			}
			parts := strings.SplitN(sections[0], ".", 2)
			resource.Resource = parts[0]
			if len(parts) == 2 {
				resource.Group = parts[1]
			}
			if resource.Resource == "" {
				return create.ResourceOptions{}, cli.Rejection.Message("rbac-resource", "resource must be specified", cli.Rejection.At(0, 0, len(value)))
			}
			return resource, nil
		},
		Format: func(value create.ResourceOptions) string {
			resource := value.Resource
			if value.Group != "" {
				resource += "." + value.Group
			}
			if value.SubResource != "" {
				resource += "/" + value.SubResource
			}
			return resource
		},
		Snapshot: func(value create.ResourceOptions) create.ResourceOptions { return value },
	}
	return codec
}

func uniqueCreateStrings(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

func uniqueCreateWildcardStrings(values []string) []string {
	for _, value := range values {
		if value == "*" {
			return []string{"*"}
		}
	}
	return uniqueCreateStrings(values)
}

func normalizeCreateRoleResources(values []create.ResourceOptions) []create.ResourceOptions {
	for _, value := range values {
		if value.Resource == "*" && value.Group == "" && value.SubResource == "" {
			return []create.ResourceOptions{value}
		}
	}
	return append([]create.ResourceOptions(nil), values...)
}

type createRoleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
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

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			resourceInput := cb.Values.Sequence(&cb.Config.ResourceInputs, createRoleResourceLanguage())
			resources := cb.Values.Derive(&cb.Config.Resources, cli.Value.ComputedBy(func(_ context.Context, _ *createRoleConfig, dependencies cli.Dependencies) ([]create.ResourceOptions, error) {
				values, _ := dependencies.Get(resourceInput)
				return normalizeCreateRoleResources(values), nil
			}).From(resourceInput.OptionalDependency()))
			resourceFlag := cb.Flags.Sequence(resourceInput)
			resourceFlag = resourceFlag.DelimitedBy(",")
			resourceFlag = resourceFlag.Description("Resource that the rule applies to")
			resourceFlag.Long("resource")

			resourceNameInput := cb.Values.Sequence(&cb.Config.ResourceNameInputs, cli.Value.String())
			cb.Values.Derive(&cb.Config.ResourceNames, cli.Value.ComputedBy(func(_ context.Context, _ *createRoleConfig, dependencies cli.Dependencies) ([]string, error) {
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
			verbs := cb.Values.Derive(&cb.Config.Verbs, cli.Value.ComputedBy(func(_ context.Context, _ *createRoleConfig, dependencies cli.Dependencies) ([]string, error) {
				values, _ := dependencies.Get(verbInput)
				return uniqueCreateWildcardStrings(values), nil
			}).From(verbInput.OptionalDependency()))
			verbFlag := cb.Flags.Sequence(verbInput)
			verbFlag = verbFlag.DelimitedBy(",")
			verbFlag = verbFlag.Description("Verb that applies to the resources contained in the rule")
			verbFlag.Long("verb")

			cb.Rule.Check("kubectl-create-role-verbs", func(_ context.Context, cfg *createRoleConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Verbs) > 0, nil
			}).DependsOn(verbs)
			cb.Rule.Check("kubectl-create-role-resources", func(_ context.Context, cfg *createRoleConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Resources) > 0, nil
			}).DependsOn(resources)
		},
		Run: runCreateRole,
	}
	return cmd
}

func roleBindingServiceAccountLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "service-account-subject",
		Parse: func(value string) (string, error) {
			parts := strings.Split(value, ":")
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return "", cli.Rejection.Message("service-account-subject", "expected namespace:name", cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createRoleBindingConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	ClusterRole              string
	DryRun                   string
	FieldManager             string
	Groups                   []string
	Output                   string
	Role                     string
	SaveConfig               bool
	ServiceAccounts          []string
	ShowManagedFields        bool
	Template                 string
	Users                    []string
	Validate                 string
}

func createRoleBindingCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createRoleBindingConfig] {
	cmd := cli.Command[createRoleBindingConfig]{
		Name: "rolebinding",
		Description: cli.Description{
			Short: "Create a role binding for a particular role or cluster role",
			Long:  "Create a role binding for a particular role or cluster role.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a role binding for user1, user2, and group1 using the admin cluster role",
				CommandLine: "kubectl create rolebinding admin --clusterrole=admin --user=user1 --user=user2 --group=group1",
			},
			{
				Description: "Create a role binding for service account monitoring:sa-dev using the admin role",
				CommandLine: "kubectl create rolebinding admin-binding --role=admin --serviceaccount=monitoring:sa-dev",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createRoleBindingConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createRoleBindingConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			cb.Rule.NotEmpty(name)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			clusterRoleFlag := cb.Flags.String(&cb.Config.ClusterRole)
			clusterRoleFlag = clusterRoleFlag.Description("ClusterRole this RoleBinding should reference")
			clusterRoleFlag.Long("clusterrole")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			groupValue := cb.Values.Sequence(&cb.Config.Groups, cli.Value.String())
			groupFlag := cb.Flags.Sequence(groupValue)
			groupFlag = groupFlag.Description("Groups to bind to the role. The flag can be repeated to add multiple groups.")
			groupFlag.Long("group")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			roleFlag := cb.Flags.String(&cb.Config.Role)
			roleFlag = roleFlag.Description("Role this RoleBinding should reference")
			roleFlag.Long("role")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			serviceAccountValue := cb.Values.Sequence(&cb.Config.ServiceAccounts, roleBindingServiceAccountLanguage())
			serviceAccountFlag := cb.Flags.Sequence(serviceAccountValue)
			serviceAccountFlag = serviceAccountFlag.Description("Service accounts to bind to the role, in the format <namespace>:<name>. The flag can be repeated to add multiple service accounts.")
			serviceAccountFlag.Long("serviceaccount")

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			userValue := cb.Values.Sequence(&cb.Config.Users, cli.Value.String())
			userFlag := cb.Flags.Sequence(userValue)
			userFlag = userFlag.Description("Usernames to bind to the role. The flag can be repeated to add multiple users.")
			userFlag.Long("user")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Check("kubectl-create-rolebinding-role-reference", func(_ context.Context, cfg *createRoleBindingConfig, _ cli.Dependencies) (bool, error) {
				return (cfg.ClusterRole == "") != (cfg.Role == ""), nil
			}).
				DependsOn(clusterRoleFlag.Destination().OptionalDependency()).
				DependsOn(roleFlag.Destination().OptionalDependency())
		},
		Run: runCreateRoleBinding,
	}
	return cmd
}

func createSecretCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name: "secret",
		Description: cli.Description{
			Short: "Create a secret using a specified subcommand",
			Long:  "Create a secret with specified type.\n\n A docker-registry type secret is for accessing a container registry.\n\n A generic type secret indicate an Opaque secret type.\n\n A tls type secret holds TLS certificate and its associated key.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			dockerRegistry := createSecretDockerRegistryCommand(client)
			cb.Subcommands.Command(dockerRegistry)
			generic := createSecretGenericCommand(client)
			cb.Subcommands.Command(generic)
			tls := createSecretTLSCommand(client)
			cb.Subcommands.Command(tls)
		},
	}
	return cmd
}

func dockerConfigFileLanguage() cli.ValueCodec[string] {
	fileSource := createFileSourceLanguage()
	codec := cli.ValueCodec[string]{
		Name: "docker-config-file",
		Parse: func(value string) (string, error) {
			if !strings.Contains(value, "=") {
				value = ".dockerconfigjson=" + value
			}
			return fileSource.Parse(value)
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createSecretDockerRegistryConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	DockerEmail              string
	DockerPassword           string
	DockerServer             string
	DockerUsername           string
	DryRun                   string
	FieldManager             string
	FromFile                 []string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createSecretDockerRegistryCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createSecretDockerRegistryConfig] {
	cmd := cli.Command[createSecretDockerRegistryConfig]{
		Name: "docker-registry",
		Description: cli.Description{
			Short: "Create a secret for use with a Docker registry",
			Long:  "Create a new secret for use with Docker registries.\n        \n        Dockercfg secrets are used to authenticate against Docker registries.\n        \n        When using the Docker command line to push images, you can authenticate to a given registry by running:\n        '$ docker login DOCKER_REGISTRY_SERVER --username=DOCKER_USER --password=DOCKER_PASSWORD --email=DOCKER_EMAIL'.\n        \n That produces a ~/.dockercfg file that is used by subsequent 'docker push' and 'docker pull' commands to authenticate to the registry. The email address is optional.\n\n        When creating applications, you may have a Docker registry that requires authentication.  In order for the\n        nodes to pull images on your behalf, they must have the credentials.  You can provide this information\n        by creating a dockercfg secret and attaching it to your service account.",
		},
		Examples: []cli.Example{
			{
				Description: "If you do not already have a .dockercfg file, create a dockercfg secret directly",
				CommandLine: "kubectl create secret docker-registry my-secret --docker-server=DOCKER_REGISTRY_SERVER --docker-username=DOCKER_USER --docker-password=DOCKER_PASSWORD --docker-email=DOCKER_EMAIL",
			},
			{
				Description: "Create a new secret named my-secret from ~/.docker/config.json",
				CommandLine: "kubectl create secret docker-registry my-secret --from-file=path/to/.docker/config.json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createSecretDockerRegistryConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretDockerRegistryConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretDockerRegistryConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			cb.Rule.NotEmpty(name)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			appendHashFlag = appendHashFlag.Description("Append a hash of the secret to its name.")
			appendHashFlag.Long("append-hash").ImplicitValue(true)

			dockerEmailFlag := cb.Flags.String(&cb.Config.DockerEmail)
			dockerEmailFlag = dockerEmailFlag.Description("Email for Docker registry")
			dockerEmailFlag.Long("docker-email")

			dockerPasswordFlag := cb.Flags.String(&cb.Config.DockerPassword)
			dockerPasswordFlag = dockerPasswordFlag.Description("Password for Docker registry authentication")
			dockerPasswordFlag.Long("docker-password")

			dockerServerFlag := cb.Flags.String(&cb.Config.DockerServer)
			dockerServerFlag = dockerServerFlag.Description("Server location for Docker registry")
			cb.Values.Default(dockerServerFlag, "https://index.docker.io/v1/")
			dockerServerFlag.Long("docker-server")

			dockerUsernameFlag := cb.Flags.String(&cb.Config.DockerUsername)
			dockerUsernameFlag = dockerUsernameFlag.Description("Username for Docker registry authentication")
			dockerUsernameFlag.Long("docker-username")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, dockerConfigFileLanguage())
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag = fromFileFlag.DelimitedBy(",")
			fromFileFlag = fromFileFlag.Description("Key files can be specified using their file path, in which case a default name of .dockerconfigjson will be given to them, or optionally with a name and file path, in which case the given name will be used. Specifying a directory will iterate each named file in the directory that is a valid secret key. For this command, the key should always be .dockerconfigjson.")
			fromFileFlag.Long("from-file")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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

			cb.Rule.Check("kubectl-create-secret-docker-registry-source", func(_ context.Context, cfg *createSecretDockerRegistryConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.FromFile) > 0 || (cfg.DockerUsername != "" && cfg.DockerPassword != "" && cfg.DockerServer != ""), nil
			}).
				DependsOn(fromFileValue.OptionalDependency()).
				DependsOn(dockerUsernameFlag.Destination().OptionalDependency()).
				DependsOn(dockerPasswordFlag.Destination().OptionalDependency()).
				DependsOn(dockerServerFlag.Destination())
		},
		Run: runCreateSecretDockerRegistry,
	}
	return cmd
}

type createSecretGenericConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	DryRun                   string
	FieldManager             string
	FromEnvFile              []string
	FromFile                 []string
	FromLiteral              []string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Type                     string
	Validate                 string
}

func createSecretGenericCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createSecretGenericConfig] {
	cmd := cli.Command[createSecretGenericConfig]{
		Name: "generic",
		Description: cli.Description{
			Short: "Create a secret from a local file, directory, or literal value",
			Long:  "Create a secret based on a file, directory, or specified literal value.\n\n A single secret may package one or more key/value pairs.\n\n When creating a secret based on a file, the key will default to the basename of the file, and the value will default to the file content. If the basename is an invalid key or you wish to chose your own, you may specify an alternate key.\n\n When creating a secret based on a directory, each file whose basename is a valid key in the directory will be packaged into the secret. Any directory entries except regular files are ignored (e.g. subdirectories, symlinks, devices, pipes, etc).",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new secret named my-secret with keys for each file in folder bar",
				CommandLine: "kubectl create secret generic my-secret --from-file=path/to/bar",
			},
			{
				Description: "Create a new secret named my-secret with specified keys instead of names on disk",
				CommandLine: "kubectl create secret generic my-secret --from-file=ssh-privatekey=path/to/id_rsa --from-file=ssh-publickey=path/to/id_rsa.pub",
			},
			{
				Description: "Create a new secret named my-secret with key1=supersecret and key2=topsecret",
				CommandLine: "kubectl create secret generic my-secret --from-literal=key1=supersecret --from-literal=key2=topsecret",
			},
			{
				Description: "Create a new secret named my-secret using a combination of a file and a literal",
				CommandLine: "kubectl create secret generic my-secret --from-file=ssh-privatekey=path/to/id_rsa --from-literal=passphrase=topsecret",
			},
			{
				Description: "Create a new secret named my-secret from env files",
				CommandLine: "kubectl create secret generic my-secret --from-env-file=path/to/foo.env --from-env-file=path/to/bar.env",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createSecretGenericConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretGenericConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretGenericConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			cb.Rule.NotEmpty(name)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			appendHashFlag = appendHashFlag.Description("Append a hash of the secret to its name.")
			appendHashFlag.Long("append-hash").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			fromEnvFileValue := cb.Values.Sequence(&cb.Config.FromEnvFile, cli.Value.String())
			fromEnvFileFlag := cb.Flags.Sequence(fromEnvFileValue)
			fromEnvFileFlag = fromEnvFileFlag.DelimitedBy(",")
			fromEnvFileFlag = fromEnvFileFlag.Description("Specify the path to a file to read lines of key=val pairs to create a secret.")
			fromEnvFileFlag.Long("from-env-file")

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, createFileSourceLanguage())
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag = fromFileFlag.DelimitedBy(",")
			fromFileFlag = fromFileFlag.Description("Key files can be specified using their file path, in which case a default name will be given to them, or optionally with a name and file path, in which case the given name will be used.  Specifying a directory will iterate each named file in the directory that is a valid secret key.")
			fromFileFlag.Long("from-file")

			fromLiteralValue := cb.Values.Sequence(&cb.Config.FromLiteral, createLiteralSourceLanguage())
			fromLiteralFlag := cb.Flags.Sequence(fromLiteralValue)
			fromLiteralFlag = fromLiteralFlag.Description("Specify a key and literal value to insert in secret (i.e. mykey=somevalue)")
			fromLiteralFlag.Long("from-literal")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			typeFlag := cb.Flags.String(&cb.Config.Type)
			typeFlag = typeFlag.Description("The type of secret to create")
			typeFlag.Long("type")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.When(cb.When.Resolved(fromEnvFileValue)).Forbid(fromFileValue)
			cb.Rule.When(cb.When.Resolved(fromEnvFileValue)).Forbid(fromLiteralValue)
		},
		Run: runCreateSecretGeneric,
	}
	return cmd
}

type createSecretTLSConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	Cert                     string
	DryRun                   string
	FieldManager             string
	Key                      string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createSecretTLSCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createSecretTLSConfig] {
	cmd := cli.Command[createSecretTLSConfig]{
		Name: "tls",
		Description: cli.Description{
			Short: "Create a TLS secret",
			Long:  "Create a TLS secret from the given public/private key pair.\n\n The public/private key pair must exist beforehand. The public key certificate must be .PEM encoded and match the given private key.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new TLS secret named tls-secret with the given key pair",
				CommandLine: "kubectl create secret tls tls-secret --cert=path/to/tls.crt --key=path/to/tls.key",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createSecretTLSConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretTLSConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createSecretTLSConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			appendHashFlag = appendHashFlag.Description("Append a hash of the secret to its name.")
			appendHashFlag.Long("append-hash").ImplicitValue(true)

			certFlag := cb.Flags.String(&cb.Config.Cert)
			certFlag = certFlag.Description("Path to PEM encoded public key certificate.")
			certFlag.Long("cert")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			keyFlag := cb.Flags.String(&cb.Config.Key)
			keyFlag = keyFlag.Description("Path to private key associated with given certificate.")
			keyFlag.Long("key")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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

			cb.Rule.Check("kubectl-create-secret-tls-key-cert", func(_ context.Context, cfg *createSecretTLSConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Key != "" && cfg.Cert != "", nil
			}).
				DependsOn(keyFlag.Destination().OptionalDependency()).
				DependsOn(certFlag.Destination().OptionalDependency())
		},
		Run: runCreateSecretTLS,
	}
	return cmd
}

func servicePortLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "service-port",
		Parse: func(value string) (string, error) {
			parts := strings.Split(value, ":")
			if len(parts) > 2 {
				return "", cli.Rejection.Message("service-port", "expected port or port:targetPort", cli.Rejection.At(0, 0, len(value)))
			}
			if _, err := net.ParsePort(parts[0], true); err != nil {
				return "", cli.Rejection.Message("service-port", err.Error(), cli.Rejection.At(0, 0, len(value)))
			}
			if len(parts) == 2 {
				if port, err := strconv.Atoi(parts[1]); err != nil {
					if problems := validation.IsValidPortName(parts[1]); len(problems) != 0 {
						return "", cli.Rejection.Message("service-port", strings.Join(problems, ","), cli.Rejection.At(0, 0, len(value)))
					}
				} else if problems := validation.IsValidPortNum(port); len(problems) != 0 {
					return "", cli.Rejection.Message("service-port", strings.Join(problems, ","), cli.Rejection.At(0, 0, len(value)))
				}
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

func createServiceCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[struct{}] {
	cmd := cli.Command[struct{}]{
		Name:    "service",
		Aliases: cli.Aliases("svc"),
		Description: cli.Description{
			Short: "Create a service using a specified subcommand",
			Long:  "Create a service using a specified subcommand.",
		},
		Declare: func(cb *cli.CommandBuilder[struct{}]) {
			clusterIP := createServiceClusterIPCommand(client)
			cb.Subcommands.Command(clusterIP)
			externalName := createServiceExternalNameCommand(client)
			cb.Subcommands.Command(externalName)
			loadBalancer := createServiceLoadBalancerCommand(client)
			cb.Subcommands.Command(loadBalancer)
			nodePort := createServiceNodePortCommand(client)
			cb.Subcommands.Command(nodePort)
		},
	}
	return cmd
}

type createServiceClusterIPConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	ClusterIP                string
	DryRun                   string
	FieldManager             string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	TCP                      []string
	Template                 string
	Validate                 string
}

func createServiceClusterIPCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createServiceClusterIPConfig] {
	cmd := cli.Command[createServiceClusterIPConfig]{
		Name: "clusterip",
		Description: cli.Description{
			Short: "Create a ClusterIP service",
			Long:  "Create a ClusterIP service with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new ClusterIP service named my-cs",
				CommandLine: "kubectl create service clusterip my-cs --tcp=5678:8080",
			},
			{
				Description: "Create a new ClusterIP service named my-cs (in headless mode)",
				CommandLine: "kubectl create service clusterip my-cs --clusterip=\"None\"",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createServiceClusterIPConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceClusterIPConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceClusterIPConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			clusterIPFlag := cb.Flags.String(&cb.Config.ClusterIP)
			clusterIPFlag = clusterIPFlag.Description("Assign your own ClusterIP or set to 'None' for a 'headless' service (no loadbalancing).")
			clusterIPFlag.Long("clusterip")

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
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			tcpValue := cb.Values.Sequence(&cb.Config.TCP, servicePortLanguage())
			tcpFlag := cb.Flags.Sequence(tcpValue)
			tcpFlag = tcpFlag.DelimitedBy(",")
			tcpFlag = tcpFlag.Description("Port pairs can be specified as '<port>:<targetPort>'.")
			tcpFlag.Long("tcp")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Check("kubectl-create-service-clusterip-ports", func(_ context.Context, cfg *createServiceClusterIPConfig, _ cli.Dependencies) (bool, error) {
				return cfg.ClusterIP == "None" || len(cfg.TCP) > 0, nil
			}).
				DependsOn(clusterIPFlag.Destination().OptionalDependency()).
				DependsOn(tcpValue.OptionalDependency())
		},
		Run: runCreateServiceClusterIP,
	}
	return cmd
}

func externalServiceNameLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "external-service-name",
		Parse: func(value string) (string, error) {
			if problems := validation.IsDNS1123Subdomain(value); len(problems) != 0 {
				return "", cli.Rejection.Message("external-service-name", strings.Join(problems, ","), cli.Rejection.At(0, 0, len(value)))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createServiceExternalNameConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	ExternalName             string
	FieldManager             string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	TCP                      []string
	Template                 string
	Validate                 string
}

func createServiceExternalNameCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createServiceExternalNameConfig] {
	cmd := cli.Command[createServiceExternalNameConfig]{
		Name: "externalname",
		Description: cli.Description{
			Short: "Create an ExternalName service",
			Long:  "Create an ExternalName service with the specified name.\n\n ExternalName service references to an external DNS address instead of only pods, which will allow application authors to reference services that exist off platform, on other clusters, or locally.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new ExternalName service named my-ns",
				CommandLine: "kubectl create service externalname my-ns --external-name bar.com",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createServiceExternalNameConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceExternalNameConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceExternalNameConfig]) {
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

			externalNameFlag := cb.Flags.Value(&cb.Config.ExternalName, externalServiceNameLanguage())
			externalNameFlag = externalNameFlag.Description("External name of service")
			externalNameFlag.Long("external-name")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			tcpValue := cb.Values.Sequence(&cb.Config.TCP, servicePortLanguage())
			tcpFlag := cb.Flags.Sequence(tcpValue)
			tcpFlag = tcpFlag.DelimitedBy(",")
			tcpFlag = tcpFlag.Description("Port pairs can be specified as '<port>:<targetPort>'.")
			tcpFlag.Long("tcp")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Require(externalNameFlag.Destination())
		},
		Run: runCreateServiceExternalName,
	}
	return cmd
}

type createServiceLoadBalancerConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	TCP                      []string
	Template                 string
	Validate                 string
}

func createServiceLoadBalancerCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createServiceLoadBalancerConfig] {
	cmd := cli.Command[createServiceLoadBalancerConfig]{
		Name: "loadbalancer",
		Description: cli.Description{
			Short: "Create a LoadBalancer service",
			Long:  "Create a LoadBalancer service with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new LoadBalancer service named my-lbs",
				CommandLine: "kubectl create service loadbalancer my-lbs --tcp=5678:8080",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createServiceLoadBalancerConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceLoadBalancerConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceLoadBalancerConfig]) {
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
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			tcpValue := cb.Values.Sequence(&cb.Config.TCP, servicePortLanguage())
			tcpFlag := cb.Flags.Sequence(tcpValue)
			tcpFlag = tcpFlag.DelimitedBy(",")
			tcpFlag = tcpFlag.Description("Port pairs can be specified as '<port>:<targetPort>'.")
			tcpFlag.Long("tcp")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Require(tcpValue)
		},
		Run: runCreateServiceLoadBalancer,
	}
	return cmd
}

type createServiceNodePortConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	NodePort                 int
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	TCP                      []string
	Template                 string
	Validate                 string
}

func createServiceNodePortCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createServiceNodePortConfig] {
	cmd := cli.Command[createServiceNodePortConfig]{
		Name: "nodeport",
		Description: cli.Description{
			Short: "Create a NodePort service",
			Long:  "Create a NodePort service with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new NodePort service named my-ns",
				CommandLine: "kubectl create service nodeport my-ns --tcp=5678:8080",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createServiceNodePortConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceNodePortConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceNodePortConfig]) {
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

			nodePortFlag := cb.Flags.Int(&cb.Config.NodePort)
			nodePortFlag = nodePortFlag.Description("Port used to expose the service on each node in a cluster.")
			nodePortFlag.Long("node-port")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			saveConfigFlag = saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			tcpValue := cb.Values.Sequence(&cb.Config.TCP, servicePortLanguage())
			tcpFlag := cb.Flags.Sequence(tcpValue)
			tcpFlag = tcpFlag.DelimitedBy(",")
			tcpFlag = tcpFlag.Description("Port pairs can be specified as '<port>:<targetPort>'.")
			tcpFlag.Long("tcp")

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			cb.Rule.Require(tcpValue)
		},
		Run: runCreateServiceNodePort,
	}
	return cmd
}

type createServiceAccountConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   string
	FieldManager             string
	Output                   string
	SaveConfig               bool
	ShowManagedFields        bool
	Template                 string
	Validate                 string
}

func createServiceAccountCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createServiceAccountConfig] {
	cmd := cli.Command[createServiceAccountConfig]{
		Name:    "serviceaccount",
		Aliases: cli.Aliases("sa"),
		Description: cli.Description{
			Short: "Create a service account with the specified name",
			Long:  "Create a service account with the specified name.",
		},
		Examples: []cli.Example{
			{
				Description: "Create a new service account named my-service-account",
				CommandLine: "kubectl create serviceaccount my-service-account",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createServiceAccountConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceAccountConfig]) {
				form.ValueFor(name, "NAME")
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createServiceAccountConfig]) {
				form.ValueFor(name, "NAME")
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

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

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

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
		},
		Run: runCreateServiceAccount,
	}
	return cmd
}

func tokenAudienceLanguage() cli.ValueCodec[string] {
	codec := cli.ValueCodec[string]{
		Name: "token-audience",
		Parse: func(value string) (string, error) {
			if value == "" {
				return "", cli.Rejection.Message("token-audience", "audience must not be empty", cli.Rejection.At(0, 0, 0))
			}
			return value, nil
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return codec
}

type createTokenConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	Audiences                []string
	BoundObjectKind          string
	BoundObjectName          string
	BoundObjectUID           string
	Duration                 time.Duration
	Output                   string
	ShowManagedFields        bool
	Template                 string
}

func createTokenCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createTokenConfig] {
	cmd := cli.Command[createTokenConfig]{
		Name: "token",
		Description: cli.Description{
			Short: "Request a service account token",
			Long:  "Request a service account token.",
		},
		Examples: []cli.Example{
			{
				Description: "Request a token to authenticate to the kube-apiserver as the service account \"myapp\" in the current namespace",
				CommandLine: "kubectl create token myapp",
			},
			{
				Description: "Request a token for a service account in a custom namespace",
				CommandLine: "kubectl create token myapp --namespace myns",
			},
			{
				Description: "Request a token with a custom expiration",
				CommandLine: "kubectl create token myapp --duration 10m",
			},
			{
				Description: "Request a token with a custom audience",
				CommandLine: "kubectl create token myapp --audience https://example.com",
			},
			{
				Description: "Request a token bound to an instance of a Secret object",
				CommandLine: "kubectl create token myapp --bound-object-kind Secret --bound-object-name mysecret",
			},
			{
				Description: "Request a token bound to an instance of a Secret object with a specific UID",
				CommandLine: "kubectl create token myapp --bound-object-kind Secret --bound-object-name mysecret --bound-object-uid 0d4691ed-659b-4935-a832-355f77ee47cc",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createTokenConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createTokenConfig]) {
				form.ValueFor(name, "NAME").
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *createTokenConfig) (clientConfig, []string) { return cfg.Client, nil },
						"serviceaccount",
					)).
					From(client)
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createTokenConfig]) {
				form.ValueFor(name, "NAME").
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *createTokenConfig) (clientConfig, []string) { return cfg.Client, nil },
						"serviceaccount",
					)).
					From(client)
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			cb.Rule.NotEmpty(name)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			audienceValue := cb.Values.Sequence(&cb.Config.Audiences, tokenAudienceLanguage())
			audienceFlag := cb.Flags.Sequence(audienceValue)
			audienceFlag = audienceFlag.Description("Audience of the requested token. If unset, defaults to requesting a token for use with the Kubernetes API server. May be repeated to request a token valid for multiple audiences.")
			audienceFlag.Long("audience")

			boundObjectKindFlag := cb.Flags.String(&cb.Config.BoundObjectKind)
			boundObjectKindFlag = boundObjectKindFlag.Description("Kind of an object to bind the token to. Supported kinds are Node, Pod, Secret. If set, --bound-object-name must be provided.")
			boundObjectKindFlag.Long("bound-object-kind")

			boundObjectNameFlag := cb.Flags.String(&cb.Config.BoundObjectName)
			boundObjectNameFlag = boundObjectNameFlag.Description("Name of an object to bind the token to. The token will expire when the object is deleted. Requires --bound-object-kind.")
			boundObjectNameFlag.Long("bound-object-name")

			boundObjectUIDFlag := cb.Flags.String(&cb.Config.BoundObjectUID)
			boundObjectUIDFlag = boundObjectUIDFlag.Description("UID of an object to bind the token to. Requires --bound-object-kind and --bound-object-name. If unset, the UID of the existing object is used.")
			boundObjectUIDFlag.Long("bound-object-uid")

			durationFlag := cb.Flags.Value(&cb.Config.Duration, durationLanguage())
			durationFlag = durationFlag.Description("Requested lifetime of the issued token. If not set or if set to 0, the lifetime will be determined by the server automatically. The server may return a token with a longer or shorter lifetime.")
			cb.Values.Default(durationFlag, time.Duration(0))
			durationFlag.Long("duration")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")

			cb.Rule.AtLeast(durationFlag.Destination(), time.Duration(0))
			cb.Rule.Check("kubectl-create-token-duration-seconds", func(_ context.Context, cfg *createTokenConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Duration%time.Second == 0, nil
			}).DependsOn(durationFlag.Destination())
			kindSet := cb.When.NotEqual(boundObjectKindFlag.Destination(), "")
			cb.Rule.When(kindSet).In(boundObjectKindFlag.Destination(), "Node", "Pod", "Secret")
			cb.Rule.When(kindSet).Require(boundObjectNameFlag.Destination())
			cb.Rule.Unless(kindSet).Forbid(boundObjectNameFlag.Destination())
			cb.Rule.Unless(kindSet).Forbid(boundObjectUIDFlag.Destination())
		},
		Run: runCreateToken,
	}
	return cmd
}

package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createRoleConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.").Long("dry-run").ImplicitValue(cmdutil.DryRunClient)
			cb.Values.Default(dryRun, cmdutil.DryRunNone)

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			resourceInput := cb.Values.Sequence(&cb.Config.ResourceInputs, createRoleResourceLanguage)
			resources := cb.Values.Derive(&cb.Config.Resources, cli.Value.ComputedBy(func(_ context.Context, _ *createRoleConfig, dependencies cli.Dependencies) ([]create.ResourceOptions, error) {
				values, _ := dependencies.Get(resourceInput)
				return normalizeCreateRoleResources(values), nil
			}).From(resourceInput.OptionalDependency()))
			resourceFlag := cb.Flags.Sequence(resourceInput)
			resourceFlag.DelimitedBy(",")
			resourceFlag.Description("Resource that the rule applies to")
			resourceFlag.Long("resource")

			resourceNameInput := cb.Values.Sequence(&cb.Config.ResourceNameInputs, cli.Value.String())
			cb.Values.Derive(&cb.Config.ResourceNames, cli.Value.ComputedBy(func(_ context.Context, _ *createRoleConfig, dependencies cli.Dependencies) ([]string, error) {
				values, _ := dependencies.Get(resourceNameInput)
				return uniqueCreateStrings(values), nil
			}).From(resourceNameInput.OptionalDependency()))
			resourceNameFlag := cb.Flags.Sequence(resourceNameInput)
			resourceNameFlag.Description("Resource in the white list that the rule applies to, repeat this flag for multiple items")
			resourceNameFlag.Long("resource-name")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")

			verbInput := cb.Values.Sequence(&cb.Config.VerbInputs, cli.Value.String())
			verbs := cb.Values.Derive(&cb.Config.Verbs, cli.Value.ComputedBy(func(_ context.Context, _ *createRoleConfig, dependencies cli.Dependencies) ([]string, error) {
				values, _ := dependencies.Get(verbInput)
				return uniqueCreateWildcardStrings(values), nil
			}).From(verbInput.OptionalDependency()))
			verbFlag := cb.Flags.Sequence(verbInput)
			verbFlag.DelimitedBy(",")
			verbFlag.Description("Verb that applies to the resources contained in the rule")
			verbFlag.Long("verb")

			cb.Rule.Check("kubectl-create-role-verbs", func(ctx context.Context, cfg *createRoleConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Verbs) > 0, nil
			}).DependsOn(verbs)
			cb.Rule.Check("kubectl-create-role-resources", func(ctx context.Context, cfg *createRoleConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Resources) > 0, nil
			}).DependsOn(resources)
		},
		Run: runCreateRole,
	}
	return cmd
}

func runCreateRole(ctx context.Context, cfg *createRoleConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewCreateRoleOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Verbs = slices.Clone(cfg.Verbs)
	options.Resources = slices.Clone(cfg.Resources)
	options.ResourceNames = slices.Clone(cfg.ResourceNames)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig
	options.OutputFormat = cfg.Output

	var err error
	options.Mapper, err = factory.ToRESTMapper()
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.RbacV1()
	options.DryRunStrategy = cfg.DryRun
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlagsWithOutputState(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	warnCreateRoleVerbs(options.ErrOut, options.Verbs)
	if err := validateCreateRoleResources(options.Mapper, options.Verbs, options.Resources); err != nil {
		return err
	}
	return options.RunCreateRole()
}

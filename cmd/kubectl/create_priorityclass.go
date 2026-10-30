package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	schedulingv1 "k8s.io/client-go/kubernetes/typed/scheduling/v1"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createPriorityClassConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	Description              string
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	GlobalDefault            bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			descriptionFlag := cb.Flags.String(&cb.Config.Description)
			cb.Values.ResolveWith(descriptionFlag, cli.Value.KeepLast())
			descriptionFlag.Description("description is an arbitrary string that usually provides guidelines on when this priority class should be used.")
			descriptionFlag.Long("description")

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

			globalDefaultFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.GlobalDefault))
			cb.Values.ResolveWith(globalDefaultFlag, cli.Value.KeepLast())
			globalDefaultFlag.Description("global-default specifies whether this PriorityClass should be considered as the default priority.")
			globalDefaultFlag.Long("global-default").ImplicitValue(true)

			preemptionPolicyFlag := cb.Flags.String(&cb.Config.PreemptionPolicy)
			cb.Values.ResolveWith(preemptionPolicyFlag, cli.Value.KeepLast())
			preemptionPolicyFlag.Description("preemption-policy is the policy for preempting pods with lower priority.")
			cb.Values.Default(preemptionPolicyFlag, "PreemptLowerPriority")
			preemptionPolicyFlag.Long("preemption-policy")

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

			valueFlag := cb.Flags.Value(&cb.Config.Value, int32Language())
			cb.Values.ResolveWith(valueFlag, cli.Value.KeepLast())
			valueFlag.Description("the value of this priority class.")
			valueFlag.Long("value")
		},
		Run: runCreatePriorityClass,
	}
	return cmd
}

func runCreatePriorityClass(ctx context.Context, cfg *createPriorityClassConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewPriorityClassOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Value = cfg.Value
	options.GlobalDefault = cfg.GlobalDefault
	options.Description = cfg.Description
	options.PreemptionPolicy = cfg.PreemptionPolicy
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = schedulingv1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
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
	return options.Run()
}

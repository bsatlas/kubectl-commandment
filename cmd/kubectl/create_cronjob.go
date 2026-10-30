package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createCronJobConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Command                  []string
	AllowMissingTemplateKeys bool
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	Image                    string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

			imageFlag := cb.Flags.String(&cb.Config.Image)
			cb.Values.ResolveWith(imageFlag, cli.Value.KeepLast())
			imageFlag.Description("Image name to run.")
			imageFlag.Required()
			cb.Rule.Require(imageFlag.Destination())
			imageFlag.Long("image")

			restartFlag := cb.Flags.String(&cb.Config.Restart)
			cb.Values.ResolveWith(restartFlag, cli.Value.KeepLast())
			restartFlag.Description("job's restart policy. supported values: OnFailure, Never")
			cb.Values.Default(restartFlag, "OnFailure")
			restartFlag.Long("restart")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			scheduleFlag := cb.Flags.String(&cb.Config.Schedule)
			cb.Values.ResolveWith(scheduleFlag, cli.Value.KeepLast())
			scheduleFlag.Description("A schedule in the Cron format the job should be run with.")
			scheduleFlag.Required()
			cb.Rule.Require(scheduleFlag.Destination())
			scheduleFlag.Long("schedule")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
			cb.Values.Default(validateFlag, "strict")
			validateFlagLong := validateFlag.Long("validate")
			validateFlagLong.ImplicitValue("strict")
		},
		Run: runCreateCronJob,
	}
	return cmd
}

func runCreateCronJob(ctx context.Context, cfg *createCronJobConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewCreateCronJobOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Image = cfg.Image
	options.Schedule = cfg.Schedule
	options.Command = slices.Clone(cfg.Command)
	options.Restart = cfg.Restart
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig
	options.Builder = factory.NewBuilder()

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.BatchV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
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

package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createJobConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Command                  []string
	AllowMissingTemplateKeys bool
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	From                     string
	Image                    string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

			fromFlag := cb.Flags.String(&cb.Config.From)
			cb.Values.ResolveWith(fromFlag, cli.Value.KeepLast())
			fromFlag.Description("The name of the resource to create a Job from (only CronJob is supported).")
			fromFlag.Long("from")

			imageFlag := cb.Flags.String(&cb.Config.Image)
			cb.Values.ResolveWith(imageFlag, cli.Value.KeepLast())
			imageFlag.Description("Image name to run.")
			imageFlag.Long("image")

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

			cb.Rule.Check("kubectl-create-job-source", func(ctx context.Context, cfg *createJobConfig, _ cli.Dependencies) (bool, error) {
				image := cfg.Image != ""
				from := cfg.From != ""
				return image != from, nil
			}).
				DependsOn(imageFlag.Destination().OptionalDependency()).
				DependsOn(fromFlag.Destination().OptionalDependency())
			cb.Rule.Check("kubectl-create-job-from-command", func(ctx context.Context, cfg *createJobConfig, _ cli.Dependencies) (bool, error) {
				return cfg.From == "" || len(cfg.Command) == 0, nil
			}).
				DependsOn(fromFlag.Destination().OptionalDependency()).
				DependsOn(command.OptionalDependency())
		},
		Run: runCreateJob,
	}
	return cmd
}

func runCreateJob(ctx context.Context, cfg *createJobConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewCreateJobOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Image = cfg.Image
	options.From = cfg.From
	options.Command = slices.Clone(cfg.Command)
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

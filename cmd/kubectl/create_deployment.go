package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createDeploymentConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Command                  []string
	AllowMissingTemplateKeys bool
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	Images                   []string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

			imageValue := cb.Values.Sequence(&cb.Config.Images, cli.Value.String())
			imageFlag := cb.Flags.Sequence(imageValue)
			imageFlag.DelimitedBy(",")
			imageFlag.Description("Image names to run. A deployment can have multiple images set for multi-container pod.")
			imageFlag.Required()
			cb.Rule.Require(imageFlag.Destination())
			imageFlag.Long("image")

			portFlag := cb.Flags.Value(&cb.Config.Port, int32Language())
			cb.Values.ResolveWith(portFlag, cli.Value.KeepLast())
			portFlag.Description("The containerPort that this deployment exposes.")
			cb.Values.Default(portFlag, int32(-1))
			portFlag.Long("port")

			replicasFlag := cb.Flags.Value(&cb.Config.Replicas, int32Language())
			cb.Values.ResolveWith(replicasFlag, cli.Value.KeepLast())
			replicasFlag.Description("Number of replicas to create. Default is 1.")
			cb.Values.Default(replicasFlag, int32(1))
			replicasFlag.Long("replicas")
			replicasFlag.Short('r').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormAssigned, cli.FlagValueFormJoined)

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

			cb.Rule.Check("kubectl-create-deployment-images-command", func(ctx context.Context, cfg *createDeploymentConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Images) <= 1 || len(cfg.Command) == 0, nil
			}).
				DependsOn(imageValue.OptionalDependency()).
				DependsOn(command.OptionalDependency())
		},
		Run: runCreateDeployment,
	}
	return cmd
}

func runCreateDeployment(ctx context.Context, cfg *createDeploymentConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewCreateDeploymentOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Images = slices.Clone(cfg.Images)
	options.Port = cfg.Port
	options.Replicas = cfg.Replicas
	options.Command = slices.Clone(cfg.Command)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.AppsV1()
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

package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

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

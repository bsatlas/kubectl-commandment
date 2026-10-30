package main

import (
	"context"
	"slices"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createConfigMapConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	FromEnvFile              []string
	FromFile                 []string
	FromLiteral              []string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			cb.Values.ResolveWith(appendHashFlag, cli.Value.KeepLast())
			appendHashFlag.Description("Append a hash of the configmap to its name.")
			appendHashFlag.Long("append-hash").ImplicitValue(true)

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

			fromEnvFileValue := cb.Values.Sequence(&cb.Config.FromEnvFile, cli.Value.String())
			fromEnvFileFlag := cb.Flags.Sequence(fromEnvFileValue)
			fromEnvFileFlag.DelimitedBy(",")
			fromEnvFileFlag.Description("Specify the path to a file to read lines of key=val pairs to create a configmap.")
			fromEnvFileFlag.Long("from-env-file")

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, createFileSourceLanguage)
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag.DelimitedBy(",")
			fromFileFlag.Description("Key file can be specified using its file path, in which case file basename will be used as configmap key, or optionally with a key and file path, in which case the given key will be used.  Specifying a directory will iterate each named file in the directory whose basename is a valid configmap key.")
			fromFileFlag.Long("from-file")

			fromLiteralValue := cb.Values.Sequence(&cb.Config.FromLiteral, createLiteralSourceLanguage)
			fromLiteralFlag := cb.Flags.Sequence(fromLiteralValue)
			fromLiteralFlag.Description("Specify a key and literal value to insert in configmap (i.e. mykey=somevalue)")
			fromLiteralFlag.Long("from-literal")

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

			cb.Rule.When(cb.When.Resolved(fromEnvFileValue)).Forbid(fromFileValue)
			cb.Rule.When(cb.When.Resolved(fromEnvFileValue)).Forbid(fromLiteralValue)
		},
		Run: runCreateConfigMap,
	}
	return cmd
}

func runCreateConfigMap(ctx context.Context, cfg *createConfigMapConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewConfigMapOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FileSources = slices.Clone(cfg.FromFile)
	options.LiteralSources = slices.Clone(cfg.FromLiteral)
	options.EnvFileSources = slices.Clone(cfg.FromEnvFile)
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
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

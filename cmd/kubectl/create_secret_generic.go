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

type createSecretGenericConfig struct {
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
			appendHashFlag.Description("Append a hash of the secret to its name.")
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
			fromEnvFileFlag.Description("Specify the path to a file to read lines of key=val pairs to create a secret.")
			fromEnvFileFlag.Long("from-env-file")

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, createFileSourceLanguage)
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag.DelimitedBy(",")
			fromFileFlag.Description("Key files can be specified using their file path, in which case a default name will be given to them, or optionally with a name and file path, in which case the given name will be used.  Specifying a directory will iterate each named file in the directory that is a valid secret key.")
			fromFileFlag.Long("from-file")

			fromLiteralValue := cb.Values.Sequence(&cb.Config.FromLiteral, createLiteralSourceLanguage)
			fromLiteralFlag := cb.Flags.Sequence(fromLiteralValue)
			fromLiteralFlag.Description("Specify a key and literal value to insert in secret (i.e. mykey=somevalue)")
			fromLiteralFlag.Long("from-literal")

			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)

			typeFlag := cb.Flags.String(&cb.Config.Type)
			cb.Values.ResolveWith(typeFlag, cli.Value.KeepLast())
			typeFlag.Description("The type of secret to create")
			typeFlag.Long("type")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			cb.Values.ResolveWith(validateFlag, cli.Value.KeepLast())
			validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
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

func runCreateSecretGeneric(ctx context.Context, cfg *createSecretGenericConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewSecretOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Type = cfg.Type
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

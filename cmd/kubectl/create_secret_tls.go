package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type createSecretTLSConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	Cert                     string
	DryRun                   cmdutil.DryRunStrategy
	FieldManager             string
	Key                      string
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
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

			certFlag := cb.Flags.String(&cb.Config.Cert)
			cb.Values.ResolveWith(certFlag, cli.Value.KeepLast())
			certFlag.Description("Path to PEM encoded public key certificate.")
			certFlag.Long("cert")

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

			keyFlag := cb.Flags.String(&cb.Config.Key)
			cb.Values.ResolveWith(keyFlag, cli.Value.KeepLast())
			keyFlag.Description("Path to private key associated with given certificate.")
			keyFlag.Long("key")

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

			cb.Rule.Check("kubectl-create-secret-tls-key-cert", func(ctx context.Context, cfg *createSecretTLSConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Key != "" && cfg.Cert != "", nil
			}).
				DependsOn(keyFlag.Destination().OptionalDependency()).
				DependsOn(certFlag.Destination().OptionalDependency())
		},
		Run: runCreateSecretTLS,
	}
	return cmd
}

func runCreateSecretTLS(ctx context.Context, cfg *createSecretTLSConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewSecretTLSOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Key = cfg.Key
	options.Cert = cfg.Cert
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
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

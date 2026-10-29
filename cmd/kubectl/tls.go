package main

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

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

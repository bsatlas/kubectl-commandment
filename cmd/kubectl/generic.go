package main

import (
	"github.com/abstractionsinc/commandment/cli"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createSecretGenericConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	AppendHash               bool
	DryRun                   string
	FieldManager             string
	FromEnvFile              []string
	FromFile                 []string
	FromLiteral              []string
	Output                   string
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

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			appendHashFlag = appendHashFlag.Description("Append a hash of the secret to its name.")
			appendHashFlag.Long("append-hash").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			fieldManagerFlag = fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-create")
			fieldManagerFlag.Long("field-manager")

			fromEnvFileValue := cb.Values.Sequence(&cb.Config.FromEnvFile, cli.Value.String())
			fromEnvFileFlag := cb.Flags.Sequence(fromEnvFileValue)
			fromEnvFileFlag = fromEnvFileFlag.DelimitedBy(",")
			fromEnvFileFlag = fromEnvFileFlag.Description("Specify the path to a file to read lines of key=val pairs to create a secret.")
			fromEnvFileFlag.Long("from-env-file")

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, createFileSourceLanguage())
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag = fromFileFlag.DelimitedBy(",")
			fromFileFlag = fromFileFlag.Description("Key files can be specified using their file path, in which case a default name will be given to them, or optionally with a name and file path, in which case the given name will be used.  Specifying a directory will iterate each named file in the directory that is a valid secret key.")
			fromFileFlag.Long("from-file")

			fromLiteralValue := cb.Values.Sequence(&cb.Config.FromLiteral, createLiteralSourceLanguage())
			fromLiteralFlag := cb.Flags.Sequence(fromLiteralValue)
			fromLiteralFlag = fromLiteralFlag.Description("Specify a key and literal value to insert in secret (i.e. mykey=somevalue)")
			fromLiteralFlag.Long("from-literal")

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

			typeFlag := cb.Flags.String(&cb.Config.Type)
			typeFlag = typeFlag.Description("The type of secret to create")
			typeFlag.Long("type")

			validateFlag := cb.Flags.String(&cb.Config.Validate)
			validateFlag = validateFlag.Description("Must be one of: strict (or true), warn, ignore (or false). \"true\" or \"strict\" will use a schema to validate the input and fail the request if invalid. It will perform server side validation if ServerSideFieldValidation is enabled on the api-server, but will fall back to less reliable client-side validation if not. \"warn\" will warn about unknown or duplicate fields without blocking the request if server-side field validation is enabled on the API server, and behave as \"ignore\" otherwise. \"false\" or \"ignore\" will not perform any schema validation, silently dropping any unknown or duplicate fields.")
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

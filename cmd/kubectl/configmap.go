package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createConfigMapConfig struct {
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

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			appendHashFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendHash))
			appendHashFlag = appendHashFlag.Description("Append a hash of the configmap to its name.")
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
			fromEnvFileFlag = fromEnvFileFlag.Description("Specify the path to a file to read lines of key=val pairs to create a configmap.")
			fromEnvFileFlag.Long("from-env-file")

			fromFileValue := cb.Values.Sequence(&cb.Config.FromFile, cli.Value.String())
			fromFileFlag := cb.Flags.Sequence(fromFileValue)
			fromFileFlag = fromFileFlag.DelimitedBy(",")
			fromFileFlag = fromFileFlag.Description("Key file can be specified using its file path, in which case file basename will be used as configmap key, or optionally with a key and file path, in which case the given key will be used.  Specifying a directory will iterate each named file in the directory whose basename is a valid configmap key.")
			fromFileFlag.Long("from-file")

			fromLiteralValue := cb.Values.Sequence(&cb.Config.FromLiteral, cli.Value.String())
			fromLiteralFlag := cb.Flags.Sequence(fromLiteralValue)
			fromLiteralFlag = fromLiteralFlag.Description("Specify a key and literal value to insert in configmap (i.e. mykey=somevalue)")
			fromLiteralFlag.Long("from-literal")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

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
		},
		Run: runCreateConfigMap,
	}
	return cmd
}

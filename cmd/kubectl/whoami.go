package main

import (
	"context"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/auth"
)

type authWhoamiConfig struct {
	Client                   clientConfig
	AllowMissingTemplateKeys bool
	Output                   string
	ShowManagedFields        bool
	Template                 string
	OutputState              cli.OptionalValue[cli.ValueState]
	Stdout                   io.Writer
	Stderr                   io.Writer
}

func authWhoamiCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[authWhoamiConfig] {
	cmd := cli.Command[authWhoamiConfig]{
		Name: "whoami",
		Description: cli.Description{
			Short: "Experimental: Check self subject attributes",
			Long:  "Experimental: Check who you are and your attributes (groups, extra).\n\n        This command is helpful to get yourself aware of the current user attributes,\n        especially when dynamic authentication, e.g., token webhook, auth proxy, or OIDC provider,\n        is enabled in the Kubernetes cluster.",
		},
		Examples: []cli.Example{
			{
				Description: "Get your subject attributes",
				CommandLine: "kubectl auth whoami",
			},
			{
				Description: "Get your subject attributes in JSON format",
				CommandLine: "kubectl auth whoami -o json",
			},
		},
		Declare: func(cb *cli.CommandBuilder[authWhoamiConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			allowMissing := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissing, true)
			cb.Flags.ValueFor(allowMissing).Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)

			output := cb.Values.String(&cb.Config.Output)
			outputFlag := cb.Flags.ValueFor(output).Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')
			cb.Access.ValueState(output, &cb.Config.OutputState)

			showManagedFields := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFields.Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)

			template := cb.Flags.String(&cb.Config.Template)
			template.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").Long("template")
		},
		Run: runAuthWhoami,
	}
	return cmd
}

func runAuthWhoami(ctx context.Context, cfg *authWhoamiConfig) error {
	flags := auth.NewWhoAmIFlags(restClientGetter(cfg.Client), genericiooptions.IOStreams{
		Out:    cfg.Stdout,
		ErrOut: cfg.Stderr,
	})
	*flags.PrintFlags.OutputFormat = cfg.Output
	flags.PrintFlags.OutputFlagSpecified = func() bool { return flagProvided(cfg.OutputState) }
	flags.PrintFlags.JSONYamlPrintFlags.ShowManagedFields = cfg.ShowManagedFields
	*flags.PrintFlags.TemplatePrinterFlags.TemplateArgument = cfg.Template
	*flags.PrintFlags.TemplatePrinterFlags.AllowMissingKeys = cfg.AllowMissingTemplateKeys
	options, err := flags.ToOptions(ctx, nil)
	if err != nil {
		return err
	}
	return options.Run()
}

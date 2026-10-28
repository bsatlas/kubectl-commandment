package main

import (
	"context"
	"errors"
	"io"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/kubectl/pkg/scheme"
)

type configViewConfig struct {
	Kubeconfig               string
	Context                  string
	AllowMissingTemplateKeys bool
	Flatten                  bool
	Merge                    bool
	Minify                   bool
	Output                   string
	Raw                      bool
	ShowManagedFields        bool
	Template                 string
	MergeState               cli.OptionalValue[cli.ValueState]
	OutputState              cli.OptionalValue[cli.ValueState]
	Stdout                   io.Writer
}

func configViewCommand(kubeconfig, contextName cli.Destination[rootConfig, string]) cli.Command[configViewConfig] {
	cmd := cli.Command[configViewConfig]{
		Name: "view",
		Description: cli.Description{
			Short: "Display merged kubeconfig settings or a specified kubeconfig file",
			Long:  "Display merged kubeconfig settings or a specified kubeconfig file.\n\n You can use --output jsonpath={...} to extract specific values using a jsonpath expression.",
		},
		Examples: []cli.Example{
			{
				Description: "Show merged kubeconfig settings",
				CommandLine: "kubectl config view",
			},
			{
				Description: "Show merged kubeconfig settings, raw certificate data, and exposed secrets",
				CommandLine: "kubectl config view --raw",
			},
			{
				Description: "Get the password for the e2e user",
				CommandLine: "kubectl config view -o jsonpath='{.users[?(@.name == \"e2e\")].user.password}'",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configViewConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Value(contextName, &cb.Config.Context)
			cb.Access.Stdout(&cb.Config.Stdout)

			allowMissing := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissing, true)
			cb.Flags.ValueFor(allowMissing).Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)

			flatten := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Flatten))
			flatten.Description("Flatten the resulting kubeconfig file into self-contained output (useful for creating portable kubeconfig files)").Long("flatten").ImplicitValue(true)

			merge := cb.Values.Bool(&cb.Config.Merge)
			cb.Values.Default(merge, true)
			cb.Flags.ValueFor(merge).Description("Merge the full hierarchy of kubeconfig files").Long("merge").ImplicitValue(true)
			cb.Access.ValueState(merge, &cb.Config.MergeState)

			minify := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Minify))
			minify.Description("Remove all information not used by current-context from the output").Long("minify").ImplicitValue(true)

			output := cb.Values.String(&cb.Config.Output)
			cb.Values.Default(output, "yaml")
			outputFlag := cb.Flags.ValueFor(output).Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')
			cb.Access.ValueState(output, &cb.Config.OutputState)

			raw := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Raw))
			raw.Description("Display raw byte data and sensitive data").Long("raw").ImplicitValue(true)

			showManagedFields := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFields.Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)

			template := cb.Flags.String(&cb.Config.Template)
			template.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").Long("template")
		},
		Run: runConfigView,
	}
	return cmd
}

func runConfigView(_ context.Context, cfg *configViewConfig) error {
	access := configAccess(cfg.Kubeconfig)
	merge := cfg.Merge
	if access.IsExplicitFile() && !flagProvided(cfg.MergeState) {
		merge = false
	}
	if !merge && !access.IsExplicitFile() {
		return errors.New("if merge==false a precise file must be specified")
	}

	var config *api.Config
	var err error
	if merge {
		config, err = access.GetStartingConfig()
	} else {
		config, err = clientcmd.LoadFromFile(access.GetExplicitFile())
	}
	if err != nil {
		return err
	}

	if cfg.Minify {
		if cfg.Context != "" {
			config.CurrentContext = cfg.Context
		}
		if err := api.MinifyConfig(config); err != nil {
			return err
		}
	}
	if cfg.Flatten {
		if err := api.FlattenConfig(config); err != nil {
			return err
		}
	} else if !cfg.Raw {
		if err := api.RedactSecrets(config); err != nil {
			return err
		}
		api.ShortenConfig(config)
	}

	flags := genericclioptions.NewPrintFlags("").WithTypeSetter(scheme.Scheme).WithDefaultOutput("yaml")
	*flags.OutputFormat = cfg.Output
	flags.OutputFlagSpecified = func() bool { return flagProvided(cfg.OutputState) }
	flags.JSONYamlPrintFlags.ShowManagedFields = cfg.ShowManagedFields
	*flags.TemplatePrinterFlags.TemplateArgument = cfg.Template
	*flags.TemplatePrinterFlags.AllowMissingKeys = cfg.AllowMissingTemplateKeys
	printer, err := flags.ToPrinter()
	if err != nil {
		return err
	}

	object, err := latest.Scheme.ConvertToVersion(config, latest.ExternalVersion)
	if err != nil {
		return err
	}
	return printer.PrintObj(object, cfg.Stdout)
}

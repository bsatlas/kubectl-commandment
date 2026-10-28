package kubectl

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type kubercViewConfig struct {
	Streams                  genericiooptions.IOStreams
	AllowMissingTemplateKeys bool
	Kuberc                   string
	Output                   string
	ShowManagedFields        bool
	Template                 string
}

var kubercViewCommand = cli.Command[kubercViewConfig]{
	Name: "view",
	Description: cli.Description{
		Short: "Display the current kuberc configuration",
		Long:  "Display the contents of the kuberc file in the specified output format.",
	},
	Examples: []cli.Example{
		{
			Description: "View kuberc configuration in YAML format (default)",
			CommandLine: "kubectl kuberc view",
		},
		{
			Description: "View kuberc configuration in JSON format",
			CommandLine: "kubectl kuberc view --output json",
		},
		{
			Description: "View a specific kuberc file",
			CommandLine: "kubectl kuberc view --kuberc /path/to/kuberc",
		},
	},
	Declare: func(cb *cli.CommandBuilder[kubercViewConfig]) {
		cb.Access.Stdin(&cb.Config.Streams.In)
		cb.Access.Stdout(&cb.Config.Streams.Out)
		cb.Access.Stderr(&cb.Config.Streams.ErrOut)

		allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
		allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
		cb.Values.Default(allowMissingTemplateKeysFlag, true)
		allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

		kubercFlag := cb.Flags.String(&cb.Config.Kuberc)
		kubercFlag = kubercFlag.Description("Path to the kuberc file to use for preferences. This can be disabled by exporting KUBECTL_KUBERC=false feature gate or turning off the feature KUBERC=off.")
		kubercFlag.Long("kuberc")

		outputFlag := cb.Flags.String(&cb.Config.Output)
		outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
		cb.Values.Default(outputFlag, "yaml")
		outputFlag.Long("output")
		outputFlag.Short('o')

		showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
		showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
		showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

		templateFlag := cb.Flags.String(&cb.Config.Template)
		templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
		templateFlag.Long("template")
	},
	Run: runKubercView,
}

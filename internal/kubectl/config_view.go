package kubectl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
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
	OutputState              cli.OptionalValue[cli.ValueState]
	Stdout                   io.Writer
}

func configViewTemplateOutput(format string) bool {
	switch format {
	case "go-template", "go-template-file", "template", "templatefile", "jsonpath", "jsonpath-as-json", "jsonpath-file":
		return true
	default:
		return false
	}
}

func configViewOutputTemplateCompatible(output, template string) bool {
	format, payload, parameterized := strings.Cut(output, "=")
	if !configViewTemplateOutput(format) {
		return true
	}
	if parameterized {
		return payload != "" && template == ""
	}
	return template != ""
}

func configViewOutputLanguage() cli.ValueCodec[string] {
	language := cli.ValueCodec[string]{
		Name: "kubectl-config-view-output-format",
		Parse: func(value string) (string, error) {
			if value == "" {
				return value, nil
			}
			normalized := strings.ToLower(value)
			switch normalized {
			case "json", "yaml", "kyaml", "name":
				return normalized, nil
			}

			format, payload, parameterized := strings.Cut(value, "=")
			switch format {
			case "go-template", "go-template-file", "template", "templatefile", "jsonpath", "jsonpath-as-json", "jsonpath-file":
				if !parameterized || payload != "" {
					return value, nil
				}
			}

			return "", cli.Rejection.Message(
				"config-view-output-format",
				fmt.Sprintf("unsupported output format %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
	return language
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
			kubeconfig := deriveKubeconfig(cb, &cb.Config.Kubeconfig, kubeconfig)
			cb.Access.Value(contextName, &cb.Config.Context)
			cb.Access.Stdout(&cb.Config.Stdout)

			allowMissing := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissing, true)
			cb.Flags.ValueFor(allowMissing).Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)

			flatten := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Flatten))
			flatten.Description("Flatten the resulting kubeconfig file into self-contained output (useful for creating portable kubeconfig files)").Long("flatten").ImplicitValue(true)

			merge := cb.Values.Bool(&cb.Config.Merge)
			cb.Values.ComputedDefault(merge, cli.Value.ComputedBy(func(_ context.Context, _ *configViewConfig, dependencies cli.Dependencies) (bool, error) {
				path, ok := dependencies.Get(kubeconfig)
				if !ok {
					return false, errors.New("kubeconfig dependency is unavailable")
				}
				return path == "", nil
			}).From(kubeconfig))
			cb.Flags.ValueFor(merge).Description("Merge the full hierarchy of kubeconfig files").Long("merge").ImplicitValue(true)
			cb.Rule.Check("kubectl-config-view-merge", func(_ context.Context, cfg *configViewConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Merge || cfg.Kubeconfig != "", nil
			}).
				DependsOn(merge.OptionalDependency()).
				DependsOn(kubeconfig.OptionalDependency())

			minify := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Minify))
			minify.Description("Remove all information not used by current-context from the output").Long("minify").ImplicitValue(true)

			output := cb.Values.Value(&cb.Config.Output, configViewOutputLanguage())
			cb.Values.Default(output, "yaml")
			outputFlag := cb.Flags.ValueFor(output).Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')
			cb.Access.ValueState(output, &cb.Config.OutputState)

			template := cb.Flags.String(&cb.Config.Template)
			template.
				Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").
				CompleteWith(cli.CompletionNativeFiles).
				Long("template")
			cb.Rule.When(cb.Rule.ContributedBy(output, cli.ValueProducerKindFlag)).
				Check("kubectl-config-view-output-template", func(_ context.Context, cfg *configViewConfig, _ cli.Dependencies) (bool, error) {
					return configViewOutputTemplateCompatible(cfg.Output, cfg.Template), nil
				}).
				Subject(output).
				DependsOn(template.OptionalDependency())

			raw := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Raw))
			raw.Description("Display raw byte data and sensitive data").Long("raw").ImplicitValue(true)

			showManagedFields := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFields.Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)
		},
		Run: runConfigView,
	}
	return cmd
}

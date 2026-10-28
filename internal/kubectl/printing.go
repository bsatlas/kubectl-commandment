package kubectl

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

var printOutputFormats = []string{
	"json",
	"yaml",
	"kyaml",
	"name",
	"go-template",
	"go-template-file",
	"template",
	"templatefile",
	"jsonpath",
	"jsonpath-as-json",
	"jsonpath-file",
}

var printOutputLanguage = cli.ValueCodec[string]{
	Name: "kubectl-print-output",
	Parse: func(value string) (string, error) {
		if value == "" {
			return value, nil
		}
		format, payload, assigned := strings.Cut(value, "=")
		if printOutputFormat(format) && (!assigned || printOutputAcceptsArgument(format) && payload != "") {
			return value, nil
		}
		return "", cli.Rejection.Message(
			"output-format",
			fmt.Sprintf("unsupported output format %q", value),
			cli.Rejection.At(0, 0, len(value)),
		)
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

func printOutputFormat(format string) bool {
	switch strings.ToLower(format) {
	case "json", "yaml", "kyaml", "name":
		return true
	}
	return printOutputAcceptsArgument(format)
}

func printOutputAcceptsArgument(format string) bool {
	switch format {
	case "go-template", "go-template-file", "template", "templatefile", "jsonpath", "jsonpath-as-json", "jsonpath-file":
		return true
	default:
		return false
	}
}

func validPrintOutputTemplate(output, template string, outputSpecified bool) bool {
	if strings.Contains(output, "=") {
		return template == ""
	}
	if output == "" {
		return !outputSpecified
	}
	return !printOutputAcceptsArgument(output) || template != ""
}

func declarePrintFlags[C any](
	cb *cli.CommandBuilder[C],
	output *string,
	outputState *cli.OptionalValue[cli.ValueState],
	template *string,
	allowMissingTemplateKeys *bool,
	showManagedFields *bool,
) {
	allowMissing := cb.Values.Bool(allowMissingTemplateKeys)
	cb.Values.Default(allowMissing, true)
	cb.Flags.ValueFor(allowMissing).
		Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").
		Long("allow-missing-template-keys").
		ImplicitValue(true)

	outputValue := cb.Values.Value(output, printOutputLanguage)
	outputFlag := cb.Flags.ValueFor(outputValue).
		Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).").
		CompleteBy(completePrintOutputFormat[C])
	outputFlag.Long("output")
	outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
	cb.Access.ValueState(outputValue, outputState)

	cb.Flags.ValueFor(cb.Values.Bool(showManagedFields)).
		Description("If true, keep the managedFields when printing objects in JSON or YAML format.").
		Long("show-managed-fields").
		ImplicitValue(true)

	templateValue := cb.Values.String(template)
	templateFlag := cb.Flags.ValueFor(templateValue).
		Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].").
		CompleteWith(cli.CompletionNativeFiles)
	templateFlag.Long("template")

	declarePrintOutputTemplateRule(cb, outputFlag, templateFlag)
}

func declarePrintOutputTemplateRule[C any](cb *cli.CommandBuilder[C], output, template cli.FlagDeclaration[C, string]) {
	cb.Rule.Check("kubectl-print-output-template", func(ctx context.Context, _ *C, dependencies cli.Dependencies) (bool, error) {
		outputValue, _ := dependencies.Get(output)
		templateValue, _ := dependencies.Get(template)
		state, ok := cli.ValueStateFromContext(ctx, output)
		return validPrintOutputTemplate(outputValue, templateValue, ok && explicitFlagState(state)), nil
	}).
		Subject(output).
		Subject(template).
		DependsOn(output.OptionalDependency()).
		DependsOn(template.OptionalDependency())
}

func completePrintOutputFormat[C any](_ context.Context, _ *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	candidates := make([]cli.CompletionCandidate, 0, len(printOutputFormats))
	for _, format := range printOutputFormats {
		if strings.HasPrefix(format, request.Text) {
			candidate := cli.CompletionCandidate{
				Value: format,
				Kind:  cli.CompletionCandidateValue,
			}
			candidates = append(candidates, candidate)
		}
	}
	completion := cli.CompletionContribution{
		Candidates: candidates,
	}
	return completion, nil
}

func configurePrintFlags(flags *genericclioptions.PrintFlags, output, template string, allowMissingTemplateKeys, showManagedFields bool) {
	configurePrintFlagValues(flags, output, template, allowMissingTemplateKeys, showManagedFields)
	flags.OutputFlagSpecified = func() bool { return output != "" }
}

func configurePrintFlagsWithOutputState(flags *genericclioptions.PrintFlags, output, template string, allowMissingTemplateKeys, showManagedFields bool, outputState cli.OptionalValue[cli.ValueState]) {
	configurePrintFlagValues(flags, output, template, allowMissingTemplateKeys, showManagedFields)
	flags.OutputFlagSpecified = func() bool { return explicitFlagValue(outputState) }
}

func configurePrintFlagValues(flags *genericclioptions.PrintFlags, output, template string, allowMissingTemplateKeys, showManagedFields bool) {
	if flags.OutputFormat != nil {
		*flags.OutputFormat = output
	}
	if flags.TemplatePrinterFlags != nil {
		if flags.TemplatePrinterFlags.TemplateArgument != nil {
			*flags.TemplatePrinterFlags.TemplateArgument = template
		}
		if flags.TemplatePrinterFlags.AllowMissingKeys != nil {
			*flags.TemplatePrinterFlags.AllowMissingKeys = allowMissingTemplateKeys
		}
	}
	if flags.JSONYamlPrintFlags != nil {
		flags.JSONYamlPrintFlags.ShowManagedFields = showManagedFields
	}
}

func printObject(flags *genericclioptions.PrintFlags, dryRun util.DryRunStrategy, out io.Writer) (func(runtime.Object) error, error) {
	util.PrintFlagsWithDryRunStrategy(flags, dryRun)
	printer, err := flags.ToPrinter()
	if err != nil {
		return nil, err
	}
	return func(object runtime.Object) error {
		return printer.PrintObj(object, out)
	}, nil
}

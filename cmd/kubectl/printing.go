package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

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

func validPrintOutputTemplate(output, template string) bool {
	if strings.Contains(output, "=") {
		return template == ""
	}
	return !printOutputAcceptsArgument(output) || template != ""
}

func configurePrintFlags(flags *genericclioptions.PrintFlags, output, template string, allowMissingTemplateKeys, showManagedFields bool) {
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
	flags.OutputFlagSpecified = func() bool { return output != "" }
}

func configurePrintFlagsWithOutputState(flags *genericclioptions.PrintFlags, output, template string, allowMissingTemplateKeys, showManagedFields bool, outputState cli.OptionalValue[cli.ValueState]) {
	configurePrintFlags(flags, output, template, allowMissingTemplateKeys, showManagedFields)
	flags.OutputFlagSpecified = func() bool { return explicitFlagValue(outputState) }
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

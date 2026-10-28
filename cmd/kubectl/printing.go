package main

import (
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

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

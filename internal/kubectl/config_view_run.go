package kubectl

import (
	"context"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/kubectl/pkg/scheme"
)

func runConfigView(_ context.Context, cfg *configViewConfig) error {
	access := configAccess(cfg.Kubeconfig)
	var config *api.Config
	var err error
	if cfg.Merge {
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

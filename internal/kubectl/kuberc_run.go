package kubectl

import (
	"context"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/kuberc"
)

func runKubercSet(_ context.Context, cfg *kubercSetConfig) error {
	path, _, err := kubercPath(cfg.Kuberc)
	if err != nil {
		return err
	}
	options := kuberc.NewSetOptions(cfg.Streams)
	options.KubeRCFile = path
	options.Section = cfg.Section
	options.Command = cfg.Command
	options.AliasName = cfg.Name
	options.Options = append([]string(nil), cfg.Option...)
	options.PrependArgs = append([]string(nil), cfg.Prependarg...)
	options.AppendArgs = append([]string(nil), cfg.Appendarg...)
	options.Overwrite = cfg.Overwrite
	options.PluginPolicy = cfg.Policy
	options.AllowlistEntries = append([]string(nil), cfg.AllowlistEntry...)
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runKubercView(_ context.Context, cfg *kubercViewConfig) error {
	path, explicit, err := kubercPath(cfg.Kuberc)
	if err != nil {
		return err
	}
	printFlags := genericclioptions.NewPrintFlags("").WithDefaultOutput("yaml")
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options := &kuberc.ViewOptions{
		KubeRCFile: path,
		Explicit:   explicit,
		PrintFlags: printFlags,
		IOStreams:  cfg.Streams,
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

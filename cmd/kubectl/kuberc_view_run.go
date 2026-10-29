package main

import (
	"context"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/kuberc"
)

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

package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateJob(_ context.Context, cfg *createJobConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateJobOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Image = cfg.Image
	options.From = cfg.From
	options.Command = append([]string(nil), cfg.Command...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig
	options.Builder = factory.NewBuilder()

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.BatchV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

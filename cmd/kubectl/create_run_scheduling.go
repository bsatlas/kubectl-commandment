package main

import (
	"context"

	"k8s.io/client-go/kubernetes/typed/scheduling/v1"
	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreatePriorityClass(_ context.Context, cfg *createPriorityClassConfig) error {
	factory := cfg.Client.factory()
	options := create.NewPriorityClassOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Value = cfg.Value
	options.GlobalDefault = cfg.GlobalDefault
	options.Description = cfg.Description
	options.PreemptionPolicy = cfg.PreemptionPolicy
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = v1.NewForConfig(restConfig)
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

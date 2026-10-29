package main

import (
	"context"

	policyv1 "k8s.io/client-go/kubernetes/typed/policy/v1"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreatePodDisruptionBudget(_ context.Context, cfg *createPodDisruptionBudgetConfig) error {
	factory := cfg.Client.factory()
	options := create.NewPodDisruptionBudgetOpts(cfg.Streams)
	options.Name = cfg.Name
	options.MinAvailable = cfg.MinAvailable
	options.MaxUnavailable = cfg.MaxUnavailable
	options.Selector = cfg.Selector
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = policyv1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
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

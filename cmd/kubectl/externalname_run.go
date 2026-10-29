package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateServiceExternalName(_ context.Context, cfg *createServiceExternalNameConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, "ExternalName")
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
	options.ExternalName = cfg.ExternalName
	options.FieldManager = cfg.FieldManager

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
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

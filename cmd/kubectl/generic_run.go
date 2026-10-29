package main

import (
	"context"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateSecretGeneric(_ context.Context, cfg *createSecretGenericConfig) error {
	factory := cfg.Client.factory()
	options := create.NewSecretOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Type = cfg.Type
	options.FileSources = append([]string(nil), cfg.FromFile...)
	options.LiteralSources = append([]string(nil), cfg.FromLiteral...)
	options.EnvFileSources = append([]string(nil), cfg.FromEnvFile...)
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
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

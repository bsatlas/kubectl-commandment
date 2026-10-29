package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateSecretDockerRegistry(_ context.Context, cfg *createSecretDockerRegistryConfig) error {
	factory := cfg.Client.factory()
	options := create.NewSecretDockerRegistryOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FileSources = append([]string(nil), cfg.FromFile...)
	options.Username = cfg.DockerUsername
	options.Email = cfg.DockerEmail
	options.Password = cfg.DockerPassword
	options.Server = cfg.DockerServer
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

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

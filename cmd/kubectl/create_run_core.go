package main

import (
	"context"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateConfigMap(_ context.Context, cfg *createConfigMapConfig) error {
	factory := cfg.Client.factory()
	options := create.NewConfigMapOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FileSources = append([]string(nil), cfg.FromFile...)
	options.LiteralSources = append([]string(nil), cfg.FromLiteral...)
	options.EnvFileSources = append([]string(nil), cfg.FromEnvFile...)
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runCreateSecretDockerRegistry(_ context.Context, cfg *createSecretDockerRegistryConfig) error {
	factory := cfg.Client.factory()
	options := create.NewSecretDockerRegistryOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FileSources = append([]string(nil), cfg.FromFile...)
	for index, source := range options.FileSources {
		if !strings.Contains(source, "=") {
			options.FileSources[index] = v1.DockerConfigJsonKey + "=" + source
		}
	}
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runCreateSecretTLS(_ context.Context, cfg *createSecretTLSConfig) error {
	factory := cfg.Client.factory()
	options := create.NewSecretTLSOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Key = cfg.Key
	options.Cert = cfg.Cert
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceClusterIP(_ context.Context, cfg *createServiceClusterIPConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, v1.ServiceTypeClusterIP)
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
	options.ClusterIP = cfg.ClusterIP
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceExternalName(_ context.Context, cfg *createServiceExternalNameConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, v1.ServiceTypeExternalName)
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceLoadBalancer(_ context.Context, cfg *createServiceLoadBalancerConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, v1.ServiceTypeLoadBalancer)
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceNodePort(_ context.Context, cfg *createServiceNodePortConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, v1.ServiceTypeNodePort)
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
	options.NodePort = cfg.NodePort
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

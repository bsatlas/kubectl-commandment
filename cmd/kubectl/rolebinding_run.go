package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateRoleBinding(_ context.Context, cfg *createRoleBindingConfig) error {
	factory := cfg.Client.factory()
	options := create.NewRoleBindingOptions(cfg.Streams)
	options.Name = cfg.Name
	options.ClusterRole = cfg.ClusterRole
	options.Role = cfg.Role
	options.Users = append([]string(nil), cfg.Users...)
	options.Groups = append([]string(nil), cfg.Groups...)
	options.ServiceAccounts = append([]string(nil), cfg.ServiceAccounts...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	var err error
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.RbacV1()
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

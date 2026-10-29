package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateRole(_ context.Context, cfg *createRoleConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateRoleOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Verbs = append([]string(nil), cfg.Verbs...)
	options.Resources = append([]create.ResourceOptions(nil), cfg.Resources...)
	options.ResourceNames = append([]string(nil), cfg.ResourceNames...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig
	options.OutputFormat = cfg.Output

	var err error
	options.Mapper, err = factory.ToRESTMapper()
	if err != nil {
		return err
	}
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
	warnCreateRoleVerbs(options.ErrOut, options.Verbs)
	if err := validateCreateRoleResources(options.Mapper, options.Verbs, options.Resources); err != nil {
		return err
	}
	return options.RunCreateRole()
}

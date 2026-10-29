package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateClusterRole(_ context.Context, cfg *createClusterRoleConfig) error {
	factory := cfg.Client.factory()
	role := create.NewCreateRoleOptions(cfg.Streams)
	role.Name = cfg.Name
	role.Verbs = append([]string(nil), cfg.Verbs...)
	role.Resources = append([]create.ResourceOptions(nil), cfg.Resources...)
	role.ResourceNames = append([]string(nil), cfg.ResourceNames...)
	role.CreateAnnotation = cfg.SaveConfig
	role.OutputFormat = cfg.Output

	options := &create.CreateClusterRoleOptions{
		CreateRoleOptions: role,
		NonResourceURLs:   append([]string(nil), cfg.NonResourceURLs...),
		AggregationRule:   cfg.AggregationRule,
		FieldManager:      cfg.FieldManager,
	}

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
	if len(options.Resources) > 0 {
		warnCreateRoleVerbs(options.ErrOut, options.Verbs)
		if err := validateCreateRoleResources(options.Mapper, options.Verbs, options.Resources); err != nil {
			return err
		}
	}
	return options.RunCreateRole()
}

package main

import (
	"context"
	"strings"

	"k8s.io/kubectl/pkg/cmd/create"
)

func runCreateRole(_ context.Context, cfg *createRoleConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateRoleOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Verbs = uniqueCreateWildcardStrings(cfg.Verbs)
	options.Resources = createRoleResources(cfg.Resources)
	options.ResourceNames = uniqueCreateStrings(cfg.ResourceNames)
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunCreateRole()
}

func runCreateClusterRole(_ context.Context, cfg *createClusterRoleConfig) error {
	factory := cfg.Client.factory()
	role := create.NewCreateRoleOptions(cfg.Streams)
	role.Name = cfg.Name
	role.Verbs = uniqueCreateWildcardStrings(cfg.Verbs)
	role.Resources = createRoleResources(cfg.Resources)
	role.ResourceNames = uniqueCreateStrings(cfg.ResourceNames)
	role.CreateAnnotation = cfg.SaveConfig
	role.OutputFormat = cfg.Output

	options := &create.CreateClusterRoleOptions{
		CreateRoleOptions: role,
		NonResourceURLs:   uniqueCreateStrings(cfg.NonResourceURLs),
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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunCreateRole()
}

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
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runCreateClusterRoleBinding(_ context.Context, cfg *createClusterRoleBindingConfig) error {
	factory := cfg.Client.factory()
	options := create.NewClusterRoleBindingOptions(cfg.Streams)
	options.Name = cfg.Name
	options.ClusterRole = cfg.ClusterRole
	options.Users = append([]string(nil), cfg.Users...)
	options.Groups = append([]string(nil), cfg.Groups...)
	options.ServiceAccounts = append([]string(nil), cfg.ServiceAccounts...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

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

func createRoleResources(values []string) []create.ResourceOptions {
	resources := make([]create.ResourceOptions, 0, len(values))
	for _, value := range values {
		sections := strings.SplitN(value, "/", 2)
		resource := create.ResourceOptions{}
		if len(sections) == 2 {
			resource.SubResource = sections[1]
		}
		parts := strings.SplitN(sections[0], ".", 2)
		resource.Resource = parts[0]
		if len(parts) == 2 {
			resource.Group = parts[1]
		}
		if resource.Resource == "*" && len(parts) == 1 && len(sections) == 1 {
			return []create.ResourceOptions{resource}
		}
		resources = append(resources, resource)
	}
	return resources
}

func uniqueCreateWildcardStrings(values []string) []string {
	for _, value := range values {
		if value == "*" {
			return []string{"*"}
		}
	}
	return uniqueCreateStrings(values)
}

func uniqueCreateStrings(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

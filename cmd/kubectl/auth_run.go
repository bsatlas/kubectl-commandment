package main

import (
	"context"
	"errors"

	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/auth"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

func runAuthCanI(_ context.Context, cfg *authCanIConfig) error {
	options := &auth.CanIOptions{
		AllNamespaces: cfg.AllNamespaces,
		Quiet:         cfg.Quiet,
		NoHeaders:     cfg.NoHeaders,
		Subresource:   cfg.Subresource,
		List:          cfg.List,
		IOStreams:     cfg.Streams,
	}
	if err := options.Complete(cfg.Client.factory(), cfg.Arguments); err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	if options.List {
		return options.RunAccessList()
	}
	allowed, err := options.RunAccessCheck()
	if err != nil {
		return err
	}
	if !allowed {
		return silentExit(1)
	}
	return nil
}

func runAuthReconcile(_ context.Context, cfg *authReconcileConfig) error {
	if len(cfg.Arguments) != 0 {
		return errors.New("no arguments are allowed")
	}
	factory := cfg.Client.factory()
	options := auth.NewReconcileOptions(cfg.Streams)
	options.FilenameOptions = &resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	if err := options.FilenameOptions.RequireFilenameOrKustomize(); err != nil {
		return err
	}
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	if strategy != util.DryRunNone && strategy != util.DryRunClient {
		return errors.New("--dry-run=server is not supported for auth reconcile")
	}
	options.DryRun = strategy == util.DryRunClient
	options.RemoveExtraPermissions = cfg.RemoveExtraPermissions
	options.RemoveExtraSubjects = cfg.RemoveExtraSubjects

	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	result := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, options.FilenameOptions).
		Flatten().
		Local().
		Do()
	if err := result.Err(); err != nil {
		return err
	}
	options.Visitor = result
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.RBACClient = clientset.RbacV1()
	options.NamespaceClient = clientset.CoreV1()
	if options.DryRun {
		options.PrintFlags.Complete("%s (dry run)")
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printer, err := options.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}
	options.PrintObject = printer.PrintObj
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunReconcile()
}

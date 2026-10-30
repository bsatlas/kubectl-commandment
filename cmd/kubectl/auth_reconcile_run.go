package main

import (
	"context"

	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/auth"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

func runAuthReconcile(ctx context.Context, cfg *authReconcileConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	streams := genericiooptions.IOStreams{
		In:     cfg.Stdin,
		Out:    cfg.Stdout,
		ErrOut: cfg.Stderr,
	}
	options := auth.NewReconcileOptions(streams)
	options.FilenameOptions = &resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	options.DryRun = cfg.DryRun == util.DryRunClient
	options.RemoveExtraPermissions = cfg.RemoveExtraPermissions
	options.RemoveExtraSubjects = cfg.RemoveExtraSubjects

	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace()
	addFilenameSources(builder, enforceNamespace, *options.FilenameOptions, cfg.Stdin)
	result := builder.
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
		if err := options.PrintFlags.Complete("%s (dry run)"); err != nil {
			return err
		}
	}
	configurePrintFlagsWithOutputState(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
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

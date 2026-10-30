package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

const rolloutUndoApplyManagedResourceWarning = "Warning: resource %[1]s was previously managed with '%[3]s apply'. " +
	"Rolling back will not update the %[2]s annotation, which may cause unexpected behavior on future '%[3]s apply' operations. " +
	"Consider using '%[3]s apply' with your previous configuration file instead.\n"

func runRolloutUndo(ctx context.Context, cfg *rolloutUndoConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("rolled back").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(
		printFlags,
		cfg.Output,
		cfg.Template,
		cfg.AllowMissingTemplateKeys,
		cfg.ShowManagedFields,
		cfg.OutputState,
	)
	util.PrintFlagsWithDryRunStrategy(printFlags, cfg.DryRun)

	filenameOptions := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace().
		LabelSelectorParam(cfg.Selector)
	addFilenameSources(builder, enforceNamespace, filenameOptions, cfg.Streams.In)
	result := builder.
		ResourceTypeOrNameArgs(true, cfg.Resources.Args()...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	return result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		metadata, err := meta.Accessor(info.Object)
		if err != nil {
			return err
		}
		if _, managedWithApply := metadata.GetAnnotations()[corev1.LastAppliedConfigAnnotation]; managedWithApply {
			if _, err := fmt.Fprintf(
				cfg.Streams.ErrOut,
				rolloutUndoApplyManagedResourceWarning,
				info.ObjectName(),
				corev1.LastAppliedConfigAnnotation,
				"kubectl",
			); err != nil {
				return err
			}
		}

		rollbacker, err := polymorphichelpers.RollbackerFn(factory, info.ResourceMapping())
		if err != nil {
			return err
		}
		operation, err := rollbacker.Rollback(info.Object, nil, cfg.ToRevision, cfg.DryRun)
		if err != nil {
			return err
		}
		printFlags.NamePrintFlags.Operation = operation
		printer, err := printFlags.ToPrinter()
		if err != nil {
			return err
		}
		return printer.PrintObj(info.Object, cfg.Streams.Out)
	})
}

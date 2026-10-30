package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runRolloutPause(ctx context.Context, cfg *rolloutPauseConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("paused").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(
		printFlags,
		cfg.Output,
		cfg.Template,
		cfg.AllowMissingTemplateKeys,
		cfg.ShowManagedFields,
		cfg.OutputState,
	)
	toPrinter := func(operation string) (printers.ResourcePrinter, error) {
		printFlags.NamePrintFlags.Operation = operation
		return printFlags.ToPrinter()
	}

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

	var allErrs []error
	infos, err := result.Infos()
	if err != nil {
		allErrs = append(allErrs, err)
	}
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), set.PatchFn(polymorphichelpers.ObjectPauserFn))
	if len(patches) == 0 && len(allErrs) == 0 {
		_, err := fmt.Fprintf(cfg.Streams.ErrOut, "No resources found in %s namespace.\n", namespace)
		return err
	}

	for _, patch := range patches {
		info := patch.Info
		if patch.Err != nil {
			resourceName := info.Mapping.Resource.Resource
			if group := info.Mapping.Resource.Group; group != "" {
				resourceName += "." + group
			}
			allErrs = append(allErrs, fmt.Errorf("error: %s %q %w", resourceName, info.Name, patch.Err))
			continue
		}
		if len(patch.Patch) == 0 || string(patch.Patch) == "{}" {
			printer, err := toPrinter("already paused")
			if err != nil {
				allErrs = append(allErrs, err)
				continue
			}
			if err := printer.PrintObj(info.Object, cfg.Streams.Out); err != nil {
				allErrs = append(allErrs, err)
			}
			continue
		}

		object, err := resource.NewHelper(info.Client, info.Mapping).
			WithFieldManager(cfg.FieldManager).
			Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to patch: %w", err))
			continue
		}
		info.Refresh(object, true)
		printer, err := toPrinter("paused")
		if err != nil {
			allErrs = append(allErrs, err)
			continue
		}
		if err := printer.PrintObj(info.Object, cfg.Streams.Out); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return utilerrors.NewAggregate(allErrs)
}

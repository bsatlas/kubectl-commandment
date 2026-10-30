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

type rolloutRestartPatchUnavailableError struct {
	name string
}

func (err rolloutRestartPatchUnavailableError) Error() string {
	return "failed to create patch for " + err.name + ": if restart has already been triggered within the past second, please wait before attempting to trigger another"
}

func runRolloutRestart(ctx context.Context, cfg *rolloutRestartConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("restarted").WithTypeSetter(scheme.Scheme)
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
		NamespaceParam(namespace).DefaultNamespace()
	addFilenameSources(builder, enforceNamespace, filenameOptions, cfg.Streams.In)
	result := builder.
		LabelSelectorParam(cfg.Selector).
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
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), set.PatchFn(polymorphichelpers.ObjectRestarterFn))
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
			patchError := rolloutRestartPatchUnavailableError{name: info.Name}
			allErrs = append(allErrs, patchError)
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
		printer, err := toPrinter("restarted")
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

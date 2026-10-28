package main

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/scheme"
)

func runRolloutRestart(_ context.Context, cfg *rolloutRestartConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutRestartOptions(cfg.Streams)
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filenames),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if err := options.Complete(factory, nil, cfg.Resources); err != nil {
		return err
	}

	result := options.Builder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(options.Namespace).DefaultNamespace().
		FilenameParam(options.EnforceNamespace, &options.FilenameOptions).
		LabelSelectorParam(options.LabelSelector).
		ResourceTypeOrNameArgs(true, options.Resources...).
		ContinueOnError().Latest().Flatten().Do()
	if err := result.Err(); err != nil {
		return err
	}

	var allErrs []error
	infos, err := result.Infos()
	if err != nil {
		allErrs = append(allErrs, err)
	}
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), set.PatchFn(options.Restarter))
	if len(patches) == 0 && len(allErrs) == 0 {
		fmt.Fprintf(options.ErrOut, "No resources found in %s namespace.\n", options.Namespace)
		return nil
	}
	for _, patch := range patches {
		info := patch.Info
		if patch.Err != nil {
			resourceName := info.Mapping.Resource.Resource
			if info.Mapping.Resource.Group != "" {
				resourceName += "." + info.Mapping.Resource.Group
			}
			allErrs = append(allErrs, fmt.Errorf("error: %s %q %v", resourceName, info.Name, patch.Err))
			continue
		}
		if len(patch.Patch) == 0 || string(patch.Patch) == "{}" {
			allErrs = append(allErrs, fmt.Errorf("failed to create patch for %v: if restart has already been triggered within the past second, please wait before attempting to trigger another", info.Name))
			continue
		}
		obj, err := resource.NewHelper(info.Client, info.Mapping).WithFieldManager(cfg.FieldManager).Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to patch: %v", err))
			continue
		}
		info.Refresh(obj, true)
		printer, err := options.ToPrinter("restarted")
		if err != nil {
			allErrs = append(allErrs, err)
			continue
		}
		if err := printer.PrintObj(info.Object, options.Out); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return errors.NewAggregate(allErrs)
}

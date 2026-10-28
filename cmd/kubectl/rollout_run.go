package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runRolloutHistory(_ context.Context, cfg *rolloutHistoryConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutHistoryOptions(cfg.Streams)
	options.Revision = cfg.Revision
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if err := options.Complete(factory, nil, cfg.Arguments); err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runRolloutPause(_ context.Context, cfg *rolloutPauseConfig) error {
	factory := cfg.Client.factory()
	options := &rollout.PauseOptions{
		PrintFlags: genericclioptions.NewPrintFlags("paused").WithTypeSetter(scheme.Scheme),
		IOStreams:  cfg.Streams,
	}
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if err := options.Complete(factory, nil, cfg.Arguments); err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return runRolloutPausePatches(options, cfg.FieldManager)
}

func runRolloutRestart(_ context.Context, cfg *rolloutRestartConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutRestartOptions(cfg.Streams)
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if err := options.Complete(factory, nil, cfg.Arguments); err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return runRolloutRestartPatches(options, cfg.FieldManager)
}

func runRolloutResume(_ context.Context, cfg *rolloutResumeConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutResumeOptions(cfg.Streams)
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if err := options.Complete(factory, nil, cfg.Arguments); err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return runRolloutResumePatches(options, cfg.FieldManager)
}

func runRolloutStatus(_ context.Context, cfg *rolloutStatusConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutStatusOptions(cfg.Streams)
	options.FilenameOptions = &resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	options.LabelSelector = cfg.Selector
	options.Revision = cfg.Revision
	options.Timeout = cfg.Timeout
	options.Watch = cfg.Watch
	if err := options.Complete(factory, cfg.Arguments); err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

func runRolloutUndo(_ context.Context, cfg *rolloutUndoConfig) error {
	factory := cfg.Client.factory()
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options := rollout.NewRolloutUndoOptions(cfg.Streams)
	options.Resources = append([]string(nil), cfg.Arguments...)
	options.ToRevision = cfg.ToRevision
	options.DryRunStrategy = strategy
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	options.CmdBaseName = "kubectl"
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace(); err != nil {
		return err
	}
	options.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		options.PrintFlags.NamePrintFlags.Operation = operation
		util.PrintFlagsWithDryRunStrategy(options.PrintFlags, options.DryRunStrategy)
		return options.PrintFlags.ToPrinter()
	}
	options.RESTClientGetter = factory
	options.Builder = factory.NewBuilder
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunUndo()
}

func runRolloutPausePatches(options *rollout.PauseOptions, fieldManager string) error {
	result := options.Builder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(options.Namespace).DefaultNamespace().
		LabelSelectorParam(options.LabelSelector).
		FilenameParam(options.EnforceNamespace, &options.FilenameOptions).
		ResourceTypeOrNameArgs(true, options.Resources...).
		ContinueOnError().Latest().Flatten().Do()
	if err := result.Err(); err != nil {
		return err
	}

	allErrs := []error{}
	infos, err := result.Infos()
	if err != nil {
		allErrs = append(allErrs, err)
	}
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), set.PatchFn(polymorphichelpers.ObjectPauserFn))
	if len(patches) == 0 && len(allErrs) == 0 {
		fmt.Fprintf(options.ErrOut, "No resources found in %s namespace.\n", options.Namespace)
		return nil
	}
	for _, patch := range patches {
		info := patch.Info
		if patch.Err != nil {
			allErrs = append(allErrs, rolloutPatchError(info, patch.Err))
			continue
		}
		if len(patch.Patch) == 0 || string(patch.Patch) == "{}" {
			printer, err := options.ToPrinter("already paused")
			if err != nil {
				allErrs = append(allErrs, err)
				continue
			}
			if err := printer.PrintObj(info.Object, options.Out); err != nil {
				allErrs = append(allErrs, err)
			}
			continue
		}
		obj, err := resource.NewHelper(info.Client, info.Mapping).WithFieldManager(fieldManager).Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to patch: %v", err))
			continue
		}
		info.Refresh(obj, true)
		printer, err := options.ToPrinter("paused")
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

func runRolloutRestartPatches(options *rollout.RestartOptions, fieldManager string) error {
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
	allErrs := []error{}
	infos, err := result.Infos()
	if err != nil {
		allErrs = append(allErrs, err)
	}
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), set.PatchFn(polymorphichelpers.ObjectRestarterFn))
	if len(patches) == 0 && len(allErrs) == 0 {
		fmt.Fprintf(options.ErrOut, "No resources found in %s namespace.\n", options.Namespace)
		return nil
	}
	for _, patch := range patches {
		info := patch.Info
		if patch.Err != nil {
			allErrs = append(allErrs, rolloutPatchError(info, patch.Err))
			continue
		}
		if len(patch.Patch) == 0 || string(patch.Patch) == "{}" {
			allErrs = append(allErrs, fmt.Errorf("failed to create patch for %v: if restart has already been triggered within the past second, please wait before attempting to trigger another", info.Name))
			continue
		}
		obj, err := resource.NewHelper(info.Client, info.Mapping).WithFieldManager(fieldManager).Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
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

func runRolloutResumePatches(options *rollout.ResumeOptions, fieldManager string) error {
	result := options.Builder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(options.Namespace).DefaultNamespace().
		LabelSelectorParam(options.LabelSelector).
		FilenameParam(options.EnforceNamespace, &options.FilenameOptions).
		ResourceTypeOrNameArgs(true, options.Resources...).
		ContinueOnError().Latest().Flatten().Do()
	if err := result.Err(); err != nil {
		return err
	}
	allErrs := []error{}
	infos, err := result.Infos()
	if err != nil {
		allErrs = append(allErrs, err)
	}
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), set.PatchFn(polymorphichelpers.ObjectResumerFn))
	if len(patches) == 0 && len(allErrs) == 0 {
		fmt.Fprintf(options.ErrOut, "No resources found in %s namespace.\n", options.Namespace)
		return nil
	}
	for _, patch := range patches {
		info := patch.Info
		if patch.Err != nil {
			allErrs = append(allErrs, rolloutPatchError(info, patch.Err))
			continue
		}
		if len(patch.Patch) == 0 || string(patch.Patch) == "{}" {
			printer, err := options.ToPrinter("already resumed")
			if err != nil {
				allErrs = append(allErrs, err)
				continue
			}
			if err := printer.PrintObj(info.Object, options.Out); err != nil {
				allErrs = append(allErrs, err)
			}
			continue
		}
		obj, err := resource.NewHelper(info.Client, info.Mapping).WithFieldManager(fieldManager).Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to patch: %v", err))
			continue
		}
		info.Refresh(obj, true)
		printer, err := options.ToPrinter("resumed")
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

func rolloutPatchError(info *resource.Info, err error) error {
	resourceName := info.Mapping.Resource.Resource
	if info.Mapping.Resource.Group != "" {
		resourceName += "." + info.Mapping.Resource.Group
	}
	return fmt.Errorf("error: %s %q %v", resourceName, info.Name, err)
}

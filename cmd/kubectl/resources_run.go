package main

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runSetResources(_ context.Context, cfg *setResourcesConfig) error {
	strategy := cfg.DryRun
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil && !(cfg.Local && clientcmd.IsEmptyConfig(err)) {
		return err
	}
	filename := resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filename),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		LocalParam(cfg.Local).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		Flatten()
	if !cfg.Local {
		builder.LabelSelectorParam(cfg.Selector).ResourceTypeOrNameArgs(cfg.All, cfg.Resources...).Latest()
	}
	infos, err := builder.Do().Infos()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("resource requirements updated").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl set resources"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	var allErrs []error
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), func(object runtime.Object) ([]byte, error) {
		transformed := false
		_, err := polymorphichelpers.UpdatePodSpecForObjectFn(object, func(spec *v1.PodSpec) error {
			containers := append(selectContainers(spec.Containers, cfg.Containers), selectContainers(spec.InitContainers, cfg.Containers)...)
			if len(containers) == 0 {
				allErrs = append(allErrs, fmt.Errorf("error: unable to find container named %s", cfg.Containers))
				return nil
			}
			for _, container := range containers {
				if len(cfg.Limits) > 0 && container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				for key, value := range cfg.Limits {
					container.Resources.Limits[key] = value
				}
				if len(cfg.Requests) > 0 && container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				for key, value := range cfg.Requests {
					container.Resources.Requests[key] = value
				}
				transformed = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if !transformed {
			return nil, nil
		}
		if err := recorder.Record(object); err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		}
		return runtime.Encode(scheme.DefaultJSONEncoder(), object)
	})

	for _, patch := range patches {
		info := patch.Info
		if patch.Err != nil {
			allErrs = append(allErrs, fmt.Errorf("error: %s %v", info.ObjectName(), patch.Err))
			continue
		}
		if len(patch.Patch) == 0 || string(patch.Patch) == "{}" {
			continue
		}
		if cfg.Local || strategy == util.DryRunClient {
			if err := printObj(info.Object); err != nil {
				allErrs = append(allErrs, err)
			}
			continue
		}
		actual, err := resource.NewHelper(info.Client, info.Mapping).
			DryRun(strategy == util.DryRunServer).
			WithFieldManager(cfg.FieldManager).
			Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to patch resources update to pod template %v", err))
			continue
		}
		if err := printObj(actual); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return errors.NewAggregate(allErrs)
}

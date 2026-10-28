package main

import (
	"context"
	"fmt"

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
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	if cfg.Local && strategy == util.DryRunServer {
		return fmt.Errorf("cannot specify --local and --dry-run=server - did you mean --dry-run=client?")
	}
	if cfg.All && cfg.Selector != "" {
		return fmt.Errorf("cannot set --all and --selector at the same time")
	}
	if cfg.Limits == "" && cfg.Requests == "" {
		return fmt.Errorf("you must specify an update to requests or limits (in the form of --requests/--limits)")
	}
	limits, err := parseResourceList(cfg.Limits)
	if err != nil {
		return err
	}
	requests, err := parseResourceList(cfg.Requests)
	if err != nil {
		return err
	}

	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil && !(cfg.Local && clientcmd.IsEmptyConfig(err)) {
		return err
	}
	filename := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
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
		builder = builder.LabelSelectorParam(cfg.Selector).ResourceTypeOrNameArgs(cfg.All, cfg.Arguments...).Latest()
	} else if len(cfg.Arguments) > 0 {
		return resource.LocalResourceError
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
				if cfg.Limits != "" && container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				for key, value := range limits {
					container.Resources.Limits[key] = value
				}
				if cfg.Requests != "" && container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				for key, value := range requests {
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

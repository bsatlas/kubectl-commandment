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

func runSetImage(_ context.Context, cfg *setImageConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	if cfg.All && cfg.Selector != "" {
		return fmt.Errorf("cannot set --all and --selector at the same time")
	}
	if cfg.Local && strategy == util.DryRunServer {
		return fmt.Errorf("cannot specify --local and --dry-run=server - did you mean --dry-run=client?")
	}

	resources, imageArgs, err := util.GetResourcesAndPairs(cfg.Arguments, "image")
	if err != nil {
		return err
	}
	images, _, err := util.ParsePairs(imageArgs, "image", false)
	if err != nil {
		return err
	}
	if len(resources) < 1 && len(cfg.Filename) == 0 && cfg.Kustomize == "" {
		return fmt.Errorf("one or more resources must be specified as <resource> <name> or <resource>/<name>")
	}
	if len(images) < 1 {
		return fmt.Errorf("at least one image update is required")
	}
	if len(images) > 1 {
		if _, ok := images["*"]; ok {
			return fmt.Errorf("all containers are already specified by *, but saw more than one container_name=container_image pairs")
		}
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
		builder = builder.LabelSelectorParam(cfg.Selector).ResourceTypeOrNameArgs(cfg.All, resources...).Latest()
	} else if len(resources) > 0 {
		return resource.LocalResourceError
	}
	infos, err := builder.Do().Infos()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("image updated").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl set image"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	var allErrs []error
	patches := set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), func(object runtime.Object) ([]byte, error) {
		_, err := polymorphichelpers.UpdatePodSpecForObjectFn(object, func(spec *v1.PodSpec) error {
			for name, image := range images {
				initFound := setContainerImage(spec.InitContainers, name, image)
				containerFound := setContainerImage(spec.Containers, name, image)
				if !containerFound && !initFound {
					allErrs = append(allErrs, fmt.Errorf("error: unable to find container named %q", name))
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
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
			allErrs = append(allErrs, fmt.Errorf("failed to patch image update to pod template: %v", err))
			continue
		}
		if err := printObj(actual); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return errors.NewAggregate(allErrs)
}

func setContainerImage(containers []v1.Container, name, image string) bool {
	found := false
	for index := range containers {
		if containers[index].Name == name || name == "*" {
			containers[index].Image = image
			found = true
		}
	}
	return found
}

package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

var errSetSelectorUnsupportedService = errors.New("setting a selector is only supported for Services")

func runSetSelector(ctx context.Context, cfg *setSelectorConfig) error {
	strategy := cfg.DryRun
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	filename := resource.FilenameOptions{
		Filenames: slices.Concat(cfg.Filename...),
		Recursive: cfg.Recursive,
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		LocalParam(cfg.Local).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace()
	addFilenameSources(builder, enforceNamespace, filename, cfg.Streams.In)
	builder.Flatten()
	if !cfg.Local {
		builder.ResourceTypeOrNameArgs(cfg.All, cfg.Resources...).Latest()
	}
	result := builder.Do()
	if err := result.Err(); err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("selector updated").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl set selector"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	return result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		patch := &set.Patch{
			Info: info,
		}
		if cfg.ResourceVersion != "" {
			accessor, err := meta.Accessor(info.Object)
			if err != nil {
				return err
			}
			accessor.SetResourceVersion("")
		}
		set.CalculatePatch(patch, scheme.DefaultJSONEncoder(), func(object runtime.Object) ([]byte, error) {
			if cfg.ResourceVersion != "" {
				accessor, err := meta.Accessor(object)
				if err != nil {
					return nil, err
				}
				accessor.SetResourceVersion(cfg.ResourceVersion)
			}
			service, ok := object.(*v1.Service)
			if !ok {
				return nil, errSetSelectorUnsupportedService
			}
			if len(cfg.Selector.MatchExpressions) > 0 {
				return nil, fmt.Errorf("match expression %v not supported on this object", cfg.Selector.MatchExpressions)
			}
			service.Spec.Selector = make(map[string]string, len(cfg.Selector.MatchLabels))
			for key, value := range cfg.Selector.MatchLabels {
				service.Spec.Selector[key] = value
			}
			if err := recorder.Record(object); err != nil {
				klog.V(4).Infof("error recording current command: %v", err)
			}
			return runtime.Encode(scheme.DefaultJSONEncoder(), object)
		})
		if patch.Err != nil {
			return patch.Err
		}
		if cfg.Local || strategy == util.DryRunClient {
			return printObj(info.Object)
		}
		actual, err := resource.NewHelper(info.Client, info.Mapping).
			DryRun(strategy == util.DryRunServer).
			WithFieldManager(cfg.FieldManager).
			Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			return err
		}
		return printObj(actual)
	})
}

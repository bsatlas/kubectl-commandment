package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/scale"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

type autoscaleTarget struct {
	name             string
	mapping          *meta.RESTMapping
	namespace        string
	enforceNamespace bool
}

type autoscaleDryRunMode uint8

const (
	autoscaleDryRunNone autoscaleDryRunMode = iota
	autoscaleDryRunClient
	autoscaleDryRunServer
)

func runAutoscale(ctx context.Context, cfg *autoscaleConfig) error {
	if cfg.Max < 1 {
		return fmt.Errorf("--max=MAXPODS is required and must be at least 1, max: %d", cfg.Max)
	}
	if cfg.Max < cfg.Min {
		return fmt.Errorf("--max=MAXPODS must be larger or equal to --min=MINPODS, max: %d, min: %d", cfg.Max, cfg.Min)
	}
	if cfg.CPUPercent > 0 && cfg.CPU != "" {
		return fmt.Errorf("--cpu-percent and --cpu are mutually exclusive")
	}
	if cfg.CPU != "" {
		if _, _, _, err := parseAutoscaleResource(cfg.CPU, "cpu"); err != nil {
			return err
		}
	}
	if cfg.Memory != "" {
		if _, _, _, err := parseAutoscaleResource(cfg.Memory, "memory"); err != nil {
			return err
		}
	}

	factory := cfg.Client.factory()
	dryRun, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	dryRunMode := autoscaleDryRunNone
	switch dryRun {
	case util.DryRunClient:
		dryRunMode = autoscaleDryRunClient
	case util.DryRunServer:
		dryRunMode = autoscaleDryRunServer
	}
	discovery, err := factory.ToDiscoveryClient()
	if err != nil {
		return err
	}
	resolver := scale.NewDiscoveryScaleKindResolver(discovery)
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	recorder, err := autoscaleRecorder(cfg.Record)
	if err != nil {
		return err
	}
	filename := &resource.FilenameOptions{
		Filenames: cfg.Filename,
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	result := factory.NewBuilder().
		Unstructured().
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, filename).
		ResourceTypeOrNameArgs(false, cfg.Arguments...).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	count := 0
	err = result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		mapping := info.ResourceMapping()
		gvr := mapping.GroupVersionKind.GroupVersion().WithResource(mapping.Resource.Resource)
		if _, err := resolver.ScaleForResource(gvr); err != nil {
			return fmt.Errorf("cannot autoscale a %s: %w", mapping.GroupVersionKind.Kind, err)
		}

		target := autoscaleTarget{
			name:             info.Name,
			mapping:          mapping,
			namespace:        namespace,
			enforceNamespace: enforceNamespace,
		}
		if err := runAutoscaleV2(ctx, cfg, target, dryRunMode, recorder, clientset); err != nil {
			klog.V(1).Infof("Encountered an error with the autoscaling/v2 HorizontalPodAutoscaler: %v. Falling back to try the autoscaling/v1 HorizontalPodAutoscaler", err)
			compatible, compatibilityErr := autoscaleV1Compatible(cfg)
			if !compatible {
				return fmt.Errorf("failed to create autoscaling/v2 HPA and the configuration is incompatible with autoscaling/v1: %w", compatibilityErr)
			}
			if err := runAutoscaleV1(ctx, cfg, target, dryRunMode, recorder, clientset); err != nil {
				return err
			}
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no objects passed to autoscale")
	}
	return nil
}

func autoscaleRecorder(record bool) (genericclioptions.Recorder, error) {
	flags := genericclioptions.NewRecordFlags()
	*flags.Record = record
	if record {
		if err := flags.CompleteWithChangeCause("kubectl autoscale"); err != nil {
			return nil, err
		}
	}
	return flags.ToRecorder()
}

func autoscalePrint(cfg *autoscaleConfig, operation string, mode autoscaleDryRunMode) (func(runtime.Object) error, error) {
	flags := genericclioptions.NewPrintFlags(operation).WithTypeSetter(scheme.Scheme)
	configurePrintFlags(flags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	strategy := util.DryRunNone
	switch mode {
	case autoscaleDryRunClient:
		strategy = util.DryRunClient
	case autoscaleDryRunServer:
		strategy = util.DryRunServer
	}
	util.PrintFlagsWithDryRunStrategy(flags, strategy)
	printer, err := flags.ToPrinter()
	if err != nil {
		return nil, err
	}
	return func(object runtime.Object) error {
		return printer.PrintObj(object, cfg.Streams.Out)
	}, nil
}

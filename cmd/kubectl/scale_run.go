package main

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scale"
)

func runScale(_ context.Context, cfg *scaleConfig) error {
	factory := cfg.Client.factory()
	dryRun := cfg.DryRun
	printFlags := genericclioptions.NewPrintFlags("scaled")
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	util.PrintFlagsWithDryRunStrategy(printFlags, dryRun)
	printer, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}

	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	scales, err := util.ScaleClientFn(factory)
	if err != nil {
		return err
	}
	scaler := scale.NewScaler(scales)
	recorder, err := scaleRecorder(cfg.Record)
	if err != nil {
		return err
	}

	filenameOptions := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	result := factory.NewBuilder().
		Unstructured().
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filenameOptions).
		ResourceTypeOrNameArgs(cfg.All, cfg.Arguments...).
		Flatten().
		LabelSelectorParam(cfg.Selector).
		Do()
	if err := result.Err(); err != nil {
		return err
	}
	infos, infoErr := result.Infos()
	if len(cfg.ResourceVersion) != 0 && len(infos) > 1 {
		return fmt.Errorf("cannot use --resource-version with multiple resources")
	}
	if len(infos) == 0 {
		if infoErr != nil {
			return infoErr
		}
		return fmt.Errorf("no objects passed to scale")
	}

	var precondition *scale.ScalePrecondition
	if cfg.CurrentReplicas != -1 || cfg.ResourceVersion != "" {
		precondition = &scale.ScalePrecondition{
			Size:            cfg.CurrentReplicas,
			ResourceVersion: cfg.ResourceVersion,
		}
	}
	retry := scale.NewRetryParams(time.Second, 5*time.Minute)
	var waitForReplicas *scale.RetryParams
	if cfg.Timeout != 0 && dryRun == util.DryRunNone {
		waitForReplicas = scale.NewRetryParams(time.Second, cfg.Timeout)
	}

	for _, info := range infos {
		mapping := info.ResourceMapping()
		if dryRun == util.DryRunClient {
			if err := setReplicas(info, int64(cfg.Replicas)); err != nil {
				return err
			}
			if err := printer.PrintObj(info.Object, cfg.Streams.Out); err != nil {
				return err
			}
			continue
		}

		actualSize := new(int32)
		if err := scaler.Scale(info.Namespace, info.Name, uint(cfg.Replicas), actualSize, precondition, retry, waitForReplicas, mapping.Resource, dryRun == util.DryRunServer); err != nil {
			return err
		}
		if err := setReplicas(info, int64(*actualSize)); err != nil {
			return err
		}
		mergePatch, err := recorder.MakeRecordMergePatch(info.Object)
		if err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		} else if len(mergePatch) > 0 {
			client, err := factory.UnstructuredClientForMapping(mapping)
			if err != nil {
				return err
			}
			helper := resource.NewHelper(client, mapping)
			if _, err := helper.Patch(info.Namespace, info.Name, types.MergePatchType, mergePatch, nil); err != nil {
				klog.V(4).Infof("error recording reason: %v", err)
			}
		}
		if err := printer.PrintObj(info.Object, cfg.Streams.Out); err != nil {
			return err
		}
	}
	return infoErr
}

func scaleRecorder(record bool) (genericclioptions.Recorder, error) {
	flags := genericclioptions.NewRecordFlags()
	*flags.Record = record
	if record {
		if err := flags.CompleteWithChangeCause("kubectl scale"); err != nil {
			return nil, err
		}
	}
	return flags.ToRecorder()
}

func setReplicas(info *resource.Info, replicas int64) error {
	object, ok := info.Object.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	gvk := object.GroupVersionKind()
	if gvk.Group != "apps" && gvk.Group != "" {
		return nil
	}
	return unstructured.SetNestedField(object.Object, replicas, "spec", "replicas")
}

package main

import (
	"context"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/drain"
	"k8s.io/kubectl/pkg/scheme"
)

func runDrain(ctx context.Context, cfg *drainConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	if err := cordonNodes(cfg.Client, cfg.Streams, cfg.Arguments, cfg.Selector, cfg.DryRun, nodeUnschedulable); err != nil {
		return err
	}

	factory := cfg.Client.factory()
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	infos, err := drainNodeInfos(factory.NewBuilder(), namespace, cfg.Arguments, cfg.Selector, cfg.ChunkSize)
	if err != nil {
		return err
	}

	printObject := func(operation string, info *resource.Info) error {
		flags := genericclioptions.NewPrintFlags(operation).WithTypeSetter(scheme.Scheme)
		util.PrintFlagsWithDryRunStrategy(flags, strategy)
		printer, err := flags.ToPrinter()
		if err != nil {
			return err
		}
		return printer.PrintObj(info.Object, cfg.Streams.Out)
	}
	helper := &drain.Helper{
		Ctx:                             ctx,
		Client:                          clientset,
		Force:                           cfg.Force,
		GracePeriodSeconds:              cfg.GracePeriod,
		IgnoreAllDaemonSets:             cfg.IgnoreDaemonsets,
		Timeout:                         cfg.Timeout,
		DeleteEmptyDirData:              cfg.DeleteEmptydirData,
		Selector:                        cfg.Selector,
		PodSelector:                     cfg.PodSelector,
		ChunkSize:                       cfg.ChunkSize,
		DisableEviction:                 cfg.DisableEviction,
		SkipWaitForDeleteTimeoutSeconds: cfg.SkipWaitForDeleteTimeout,
		EvictErrorRetryDelay:            5 * time.Second,
		Out:                             cfg.Streams.Out,
		ErrOut:                          cfg.Streams.ErrOut,
		DryRunStrategy:                  strategy,
	}
	warningOptions := printers.WarningPrinterOptions{
		Color: printers.AllowsColorOutput(cfg.Streams.ErrOut),
	}
	warningPrinter := printers.NewWarningPrinter(cfg.Streams.ErrOut, warningOptions)
	helper.OnPodDeletionOrEvictionFinished = func(pod *v1.Pod, usingEviction bool, failure error) {
		operation := "deleted"
		if usingEviction {
			operation = "evicted"
		}
		if failure != nil {
			operation += " failed"
		}
		flags := genericclioptions.NewPrintFlags(operation).WithTypeSetter(scheme.Scheme)
		util.PrintFlagsWithDryRunStrategy(flags, strategy)
		printer, err := flags.ToPrinter()
		if err != nil {
			fmt.Fprintf(cfg.Streams.ErrOut, "error building printer: %v\n", err)
			fmt.Fprintf(cfg.Streams.Out, "pod %s/%s %s\n", pod.Namespace, pod.Name, operation)
			return
		}
		_ = printer.PrintObj(pod, cfg.Streams.Out)
	}

	drained := sets.New[string]()
	var failures []error
	var remaining []string
	for _, info := range infos {
		list, podErrors := helper.GetPodsForDeletion(info.Name)
		if podErrors != nil {
			failure := errors.NewAggregate(podErrors)
			failures = append(failures, failure)
			remaining = append(remaining, info.Name)
			continue
		}
		if warnings := list.Warnings(); warnings != "" {
			warningPrinter.Print(warnings)
		}
		if strategy == util.DryRunClient {
			for _, pod := range list.Pods() {
				fmt.Fprintf(cfg.Streams.Out, "evicting pod %s/%s (dry run)\n", pod.Namespace, pod.Name)
			}
		} else if err := helper.DeleteOrEvictPods(list.Pods()); err != nil {
			fmt.Fprintf(cfg.Streams.ErrOut, "error: unable to drain node %q due to error: %s, continuing command...\n", info.Name, err)
			if !drained.Has(info.Name) {
				failures = append(failures, err)
				remaining = append(remaining, info.Name)
			}
			continue
		}
		drained.Insert(info.Name)
		if err := printObject("drained", info); err != nil {
			return err
		}
	}
	if len(remaining) != 0 {
		fmt.Fprintln(cfg.Streams.ErrOut, "There are pending nodes to be drained:")
		for _, name := range remaining {
			fmt.Fprintf(cfg.Streams.ErrOut, " %s\n", name)
		}
	}
	return errors.NewAggregate(failures)
}

func drainNodeInfos(builder *resource.Builder, namespace string, names []string, selector string, chunkSize int64) ([]*resource.Info, error) {
	builder = builder.
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace().
		RequestChunksOf(chunkSize).
		ResourceNames("nodes", names...).
		SingleResourceType().
		Flatten()
	if selector != "" {
		builder = builder.LabelSelectorParam(selector).ResourceTypes("nodes")
	}
	result := builder.Do()
	if err := result.Err(); err != nil {
		return nil, err
	}
	infos := make([]*resource.Info, 0)
	if err := result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		if info.Mapping.Resource.GroupResource() != (schema.GroupResource{
			Resource: "nodes",
		}) {
			return fmt.Errorf("expected resource of type node, got %q", info.Mapping.Resource)
		}
		infos = append(infos, info)
		return nil
	}); err != nil {
		return nil, err
	}
	return infos, nil
}

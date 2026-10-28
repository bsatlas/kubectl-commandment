package main

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/drain"
	"k8s.io/kubectl/pkg/scheme"
)

type nodeSchedulingState uint8

const (
	nodeSchedulable nodeSchedulingState = iota
	nodeUnschedulable
)

func runCordon(_ context.Context, cfg *cordonConfig) error {
	return cordonNodes(cfg.Client, cfg.Streams, cfg.Arguments, cfg.Selector, cfg.DryRun, nodeUnschedulable)
}

func runUncordon(_ context.Context, cfg *uncordonConfig) error {
	return cordonNodes(cfg.Client, cfg.Streams, cfg.Arguments, cfg.Selector, cfg.DryRun, nodeSchedulable)
}

func cordonNodes(client clientConfig, streams genericiooptions.IOStreams, names []string, selector, dryRun string, state nodeSchedulingState) error {
	if len(names) == 0 && selector == "" {
		return errors.New("a node name or --selector is required")
	}
	if len(names) != 0 && selector != "" {
		return errors.New("cannot specify both a node name and --selector")
	}
	strategy, err := dryRunStrategy(dryRun)
	if err != nil {
		return err
	}

	factory := client.factory()
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace().
		ResourceNames("nodes", names...).
		SingleResourceType().
		Flatten()
	if selector != "" {
		builder = factory.NewBuilder().
			WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
			NamespaceParam(namespace).DefaultNamespace().
			LabelSelectorParam(selector).
			ResourceTypes("nodes").
			SingleResourceType().
			Flatten()
	}

	infos := make([]*resource.Info, 0)
	result := builder.Do()
	if err := result.Err(); err != nil {
		return err
	}
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
		return err
	}

	operation := func(changed bool) string {
		if state == nodeUnschedulable {
			if changed {
				return "cordoned"
			}
			return "already cordoned"
		}
		if changed {
			return "uncordoned"
		}
		return "already uncordoned"
	}
	for _, info := range infos {
		helper, err := drain.NewCordonHelperFromRuntimeObject(info.Object, scheme.Scheme, info.ResourceMapping().GroupVersionKind)
		if err != nil {
			return err
		}
		changed := helper.UpdateIfRequired(state == nodeUnschedulable)
		if changed && strategy != util.DryRunClient {
			updateErr, patchErr := helper.PatchOrReplace(clientset, strategy == util.DryRunServer)
			if patchErr != nil {
				return patchErr
			}
			if updateErr != nil {
				return updateErr
			}
		}

		printFlags := genericclioptions.NewPrintFlags(operation(changed)).WithTypeSetter(scheme.Scheme)
		util.PrintFlagsWithDryRunStrategy(printFlags, strategy)
		printer, err := printFlags.ToPrinter()
		if err != nil {
			return err
		}
		if err := printer.PrintObj(info.Object, streams.Out); err != nil {
			return err
		}
	}
	return nil
}

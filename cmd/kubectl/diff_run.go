/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/openapi3"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/apply"
	"k8s.io/kubectl/pkg/cmd/diff"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/prune"
	"k8s.io/utils/exec"
)

type diffTracker struct {
	visitedUIDs       sets.Set[types.UID]
	visitedNamespaces sets.Set[string]
}

type diffPruner struct {
	mapper        meta.RESTMapper
	dynamicClient dynamic.Interface
	labelSelector string
	resources     []prune.Resource
}

func runDiff(ctx context.Context, cfg *diffConfig) error {
	filename := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	if err := filename.RequireFilenameOrKustomize(); err != nil {
		return err
	}
	if cfg.ForceConflicts && !cfg.ServerSide {
		return fmt.Errorf("--force-conflicts only works with --server-side")
	}

	factory := cfg.Client.factory()
	fieldManager := cfg.FieldManager
	if cfg.ServerSide && !explicitFlagValue(cfg.FieldManagerState) {
		fieldManager = "kubectl"
	}

	var openAPIV3Root openapi3.Root
	if !cfg.ServerSide {
		openAPIV3Client, err := factory.OpenAPIV3Client()
		if err == nil {
			openAPIV3Root = openapi3.NewRoot(openAPIV3Client)
		} else {
			klog.V(4).Infof("warning: OpenAPI V3 Patch is enabled but is unable to be loaded. Will fall back to OpenAPI V2")
		}
	}

	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return err
	}
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	var tracker *diffTracker
	var pruner *diffPruner
	if cfg.Prune {
		mapper, err := factory.ToRESTMapper()
		if err != nil {
			return err
		}
		resources, err := prune.ParseResources(mapper, cfg.PruneAllowlist)
		if err != nil {
			return err
		}
		tracker = &diffTracker{
			visitedUIDs:       sets.New[types.UID](),
			visitedNamespaces: sets.New[string](),
		}
		pruner = &diffPruner{
			mapper:        mapper,
			dynamicClient: dynamicClient,
			labelSelector: cfg.Selector,
			resources:     resources,
		}
	}

	program := diff.NewDiffOptions(cfg.Streams).Diff
	differ, err := diff.NewDiffer("LIVE", "MERGED")
	if err != nil {
		return err
	}
	defer differ.TearDown()
	printer := diff.Printer{}

	result := factory.NewBuilder().
		Unstructured().
		VisitorConcurrency(cfg.Concurrency).
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		LabelSelectorParam(cfg.Selector).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	err = result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		local := info.Object.DeepCopyObject()
		for attempt := 1; attempt <= 4; attempt++ {
			if err := info.Get(); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
				info.Object = nil
			}
			force := attempt == 4
			if force {
				klog.Warningf("Object (%v: %v) keeps changing, diffing without lock", info.Object.GetObjectKind().GroupVersionKind(), info.Name)
			}
			object := diff.InfoObject{
				LocalObj:        local,
				Info:            info,
				Encoder:         scheme.DefaultJSONEncoder(),
				OpenAPIGetter:   factory,
				OpenAPIV3Root:   openAPIV3Root,
				Force:           force,
				ServerSideApply: cfg.ServerSide,
				FieldManager:    fieldManager,
				ForceConflicts:  cfg.ForceConflicts,
				IOStreams:       cfg.Streams,
			}
			if cfg.ServerSide {
				object.OpenAPIGetter = nil
			}
			if tracker != nil {
				tracker.markVisited(info)
			}
			err = differ.Diff(object, printer, cfg.ShowManagedFields, cfg.ShowSecrets)
			if !errors.IsConflict(err) {
				break
			}
		}
		apply.WarnIfDeleting(info.Object, cfg.Streams.ErrOut)
		return err
	})
	if pruner != nil {
		objects, pruneErr := pruner.pruneAll(ctx, tracker, namespace != "")
		if pruneErr != nil {
			klog.Warningf("pruning failed and could not be evaluated err: %v", pruneErr)
		}
		for _, object := range objects {
			name, nameErr := diffObjectName(object)
			if nameErr != nil {
				klog.Warningf("pruning failed and object name could not be retrieved: %v", nameErr)
				continue
			}
			if printErr := differ.From.Print(name, object, printer); printErr != nil {
				return printErr
			}
		}
	}
	if err != nil {
		return err
	}
	if err := program.Run(differ.From.Dir.Name, differ.To.Dir.Name); err != nil {
		if exitErr, ok := err.(exec.ExitError); ok && exitErr.ExitStatus() == 1 {
			return silentExit(1)
		}
		return err
	}
	return nil
}

func (tracker *diffTracker) markVisited(info *resource.Info) {
	if info.Namespaced() {
		tracker.visitedNamespaces.Insert(info.Namespace)
	}
	metadata, err := meta.Accessor(info.Object)
	if err == nil {
		tracker.visitedUIDs.Insert(metadata.GetUID())
	}
}

func (pruner *diffPruner) pruneAll(ctx context.Context, tracker *diffTracker, namespaceSpecified bool) ([]runtime.Object, error) {
	var objects []runtime.Object
	namespaced, cluster, err := prune.GetRESTMappings(pruner.mapper, pruner.resources, namespaceSpecified)
	if err != nil {
		return objects, fmt.Errorf("error retrieving RESTMappings to prune: %v", err)
	}
	for namespace := range tracker.visitedNamespaces {
		for _, mapping := range namespaced {
			pruned, err := pruner.prune(ctx, tracker, namespace, mapping)
			if err != nil {
				return pruned, fmt.Errorf("error pruning namespaced object %v: %v", mapping.GroupVersionKind, err)
			}
			objects = append(objects, pruned...)
		}
	}
	for _, mapping := range cluster {
		pruned, err := pruner.prune(ctx, tracker, v1.NamespaceNone, mapping)
		if err != nil {
			return pruned, fmt.Errorf("error pruning nonNamespaced object %v: %v", mapping.GroupVersionKind, err)
		}
		objects = append(objects, pruned...)
	}
	return objects, nil
}

func (pruner *diffPruner) prune(ctx context.Context, tracker *diffTracker, namespace string, mapping *meta.RESTMapping) ([]runtime.Object, error) {
	list, err := pruner.dynamicClient.Resource(mapping.Resource).Namespace(namespace).List(ctx, v1.ListOptions{
		LabelSelector: pruner.labelSelector,
	})
	if err != nil {
		return nil, err
	}
	objects, err := meta.ExtractList(list)
	if err != nil {
		return nil, err
	}
	pruned := make([]runtime.Object, 0)
	for _, object := range objects {
		metadata, err := meta.Accessor(object)
		if err != nil {
			return pruned, err
		}
		if _, ok := metadata.GetAnnotations()["kubectl.kubernetes.io/last-applied-configuration"]; !ok {
			continue
		}
		if tracker.visitedUIDs.Has(metadata.GetUID()) {
			continue
		}
		pruned = append(pruned, object)
	}
	return pruned, nil
}

func diffObjectName(object runtime.Object) (string, error) {
	gvk := object.GetObjectKind().GroupVersionKind()
	metadata, err := meta.Accessor(object)
	if err != nil {
		return "", err
	}
	group := ""
	if gvk.Group != "" {
		group = fmt.Sprintf("%v.", gvk.Group)
	}
	return group + fmt.Sprintf("%v.%v.%v.%v", gvk.Version, gvk.Kind, metadata.GetNamespace(), metadata.GetName()), nil
}

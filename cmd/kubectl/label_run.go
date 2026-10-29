/*
Copyright 2014 The Kubernetes Authors.

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
	"slices"

	"gopkg.in/evanphx/json-patch.v4"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

func runLabel(_ context.Context, cfg *labelConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
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
		Unstructured().
		LocalParam(cfg.Local).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		Flatten()
	if !cfg.Local {
		builder = builder.LabelSelectorParam(cfg.Selector).
			FieldSelectorParam(cfg.FieldSelector).
			AllNamespaces(cfg.AllNamespaces).
			ResourceTypeOrNameArgs(cfg.All, cfg.Operands.Resources...).
			Latest()
	}

	one := false
	result := builder.Do().IntoSingleItemImplied(&one)
	if err := result.Err(); err != nil {
		return err
	}
	if !one && cfg.ResourceVersion != "" {
		return fmt.Errorf("--resource-version may only be used with a single resource")
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl label"); err != nil {
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
		object := info.Object
		if cfg.ResourceVersion != "" {
			accessor, err := meta.Accessor(object)
			if err != nil {
				return err
			}
			accessor.SetResourceVersion("")
		}
		oldData, err := json.Marshal(object)
		if err != nil {
			return err
		}

		added, removed, err := applyLabels(object, cfg.Overwrite, cfg.ResourceVersion, cfg.Operands.New, cfg.Operands.Remove)
		if err != nil {
			return err
		}
		operation := labelOperation(added, removed)
		outputObject := object

		if !(strategy == util.DryRunClient || cfg.Local || cfg.List) {
			accessor, err := meta.Accessor(object)
			if err != nil {
				return err
			}
			for _, label := range cfg.Operands.Remove {
				if _, ok := accessor.GetLabels()[label]; !ok {
					fmt.Fprintf(cfg.Streams.ErrOut, "label %q not found.\n", label)
				}
			}
			if err := recorder.Record(object); err != nil {
				klog.V(4).Infof("error recording current command: %v", err)
			}
			if added.Len() > 0 || removed.Len() > 0 {
				newData, err := json.Marshal(object)
				if err != nil {
					return err
				}
				patch, patchErr := jsonpatch.CreateMergePatch(oldData, newData)
				mapping := info.ResourceMapping()
				client, err := factory.UnstructuredClientForMapping(mapping)
				if err != nil {
					return err
				}
				helper := resource.NewHelper(client, mapping).DryRun(strategy == util.DryRunServer).WithFieldManager(cfg.FieldManager)
				if patchErr == nil {
					outputObject, err = helper.Patch(info.Namespace, info.Name, types.MergePatchType, patch, nil)
				} else {
					outputObject, err = helper.Replace(info.Namespace, info.Name, false, object)
				}
				if err != nil {
					return err
				}
			}
		}

		if cfg.List {
			accessor, err := meta.Accessor(outputObject)
			if err != nil {
				return err
			}
			indent := ""
			if !one {
				indent = " "
				gvks, _, err := unstructuredscheme.NewUnstructuredObjectTyper().ObjectKinds(info.Object)
				if err != nil {
					return err
				}
				fmt.Fprintf(cfg.Streams.Out, "Listing labels for %s.%s/%s:\n", gvks[0].Kind, gvks[0].Group, info.Name)
			}
			for key, value := range accessor.GetLabels() {
				fmt.Fprintf(cfg.Streams.Out, "%s%s=%s\n", indent, key, value)
			}
			return nil
		}

		printFlags := genericclioptions.NewPrintFlags(operation).WithTypeSetter(scheme.Scheme)
		configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
		printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
		if err != nil {
			return err
		}
		return printObj(outputObject)
	})
}

func applyLabels(object runtime.Object, overwrite bool, resourceVersion string, labels map[string]string, remove []string) (sets.Set[string], sets.Set[string], error) {
	accessor, err := meta.Accessor(object)
	if err != nil {
		return nil, nil, err
	}
	if !overwrite {
		var conflicts []error
		for key, value := range labels {
			if current, found := accessor.GetLabels()[key]; found && current != value {
				conflicts = append(conflicts, fmt.Errorf("'%s' already has a value (%s), and --overwrite is false", key, current))
			}
		}
		if len(conflicts) > 0 {
			return nil, nil, fmt.Errorf("%v", conflicts)
		}
	}
	objectLabels := accessor.GetLabels()
	if objectLabels == nil {
		objectLabels = make(map[string]string)
	}
	added := sets.New[string]()
	removed := sets.New[string]()
	for key, value := range labels {
		if current, ok := objectLabels[key]; !ok || current != value {
			objectLabels[key] = value
			added.Insert(key)
		}
	}
	for _, key := range remove {
		if _, ok := objectLabels[key]; ok {
			delete(objectLabels, key)
			removed.Insert(key)
		}
	}
	accessor.SetLabels(objectLabels)
	if resourceVersion != "" {
		accessor.SetResourceVersion(resourceVersion)
	}
	return added, removed, nil
}

func labelOperation(added, removed sets.Set[string]) string {
	switch {
	case added.Len() > 0 && removed.Len() > 0:
		return "modified"
	case added.Len() > 0:
		return "labeled"
	case removed.Len() > 0:
		return "unlabeled"
	default:
		return "not labeled"
	}
}

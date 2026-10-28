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
	"strings"

	"gopkg.in/evanphx/json-patch.v4"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
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
	resources, labelArgs, err := util.GetResourcesAndPairs(cfg.Arguments, "label")
	if err != nil {
		return err
	}
	newLabels, removeLabels, err := parseLabelChanges(labelArgs)
	if err != nil {
		return err
	}
	if cfg.List && cfg.Output != "" {
		return fmt.Errorf("--list and --output may not be specified together")
	}
	if cfg.All && cfg.Selector != "" {
		return fmt.Errorf("cannot set --all and --selector at the same time")
	}
	if cfg.All && cfg.FieldSelector != "" {
		return fmt.Errorf("cannot set --all and --field-selector at the same time")
	}
	if cfg.List && (len(newLabels) > 0 || len(removeLabels) > 0) {
		return fmt.Errorf("cannot modify labels when --list is specified")
	}
	if cfg.Local {
		if strategy == util.DryRunServer {
			return fmt.Errorf("cannot specify --local and --dry-run=server - did you mean --dry-run=client?")
		}
		if len(resources) > 0 {
			return fmt.Errorf("can only use local files by -f pod.yaml or --filename=pod.json when --local=true is set")
		}
		if len(cfg.Filename) == 0 && cfg.Kustomize == "" {
			return fmt.Errorf("one or more files must be specified as -f pod.yaml or --filename=pod.json")
		}
	} else if len(resources) == 0 && len(cfg.Filename) == 0 && cfg.Kustomize == "" {
		return fmt.Errorf("one or more resources must be specified as <resource> <name> or <resource>/<name>")
	}
	if len(newLabels) == 0 && len(removeLabels) == 0 && !cfg.List {
		return fmt.Errorf("at least one label update is required")
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
			ResourceTypeOrNameArgs(cfg.All, resources...).
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

		added, removed, err := applyLabels(object, cfg.Overwrite, cfg.ResourceVersion, newLabels, removeLabels)
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
			for _, label := range removeLabels {
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

func parseLabelChanges(specs []string) (map[string]string, []string, error) {
	labels := map[string]string{}
	var remove []string
	for _, spec := range specs {
		if strings.Contains(spec, "=") {
			parts := strings.Split(spec, "=")
			if len(parts) != 2 {
				return nil, nil, fmt.Errorf("invalid label spec: %v", spec)
			}
			if errs := validation.IsValidLabelValue(parts[1]); len(errs) != 0 {
				return nil, nil, fmt.Errorf("invalid label value: %q: %s", spec, strings.Join(errs, ";"))
			}
			labels[parts[0]] = parts[1]
			continue
		}
		if strings.HasSuffix(spec, "-") {
			remove = append(remove, strings.TrimSuffix(spec, "-"))
			continue
		}
		return nil, nil, fmt.Errorf("unknown label spec: %v", spec)
	}
	for _, key := range remove {
		if _, found := labels[key]; found {
			return nil, nil, fmt.Errorf("can not both modify and remove a label in the same command")
		}
	}
	return labels, remove, nil
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
		objectLabels = map[string]string{}
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

/*
Copyright 2016 The Kubernetes Authors.

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
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/scheme"
)

func runTaint(_ context.Context, cfg *taintConfig) error {
	strategy := cfg.DryRun
	validationDirective, err := validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	factory := cfg.Client.factory()
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		return err
	}
	request, _, err := explain.SplitAndParseResourceRequest(strings.ToLower(cfg.Operands.Resources[0]), mapper)
	if err != nil {
		return err
	}
	kind, err := mapper.KindFor(request)
	if err != nil {
		return err
	}
	if kind.Kind != "Node" {
		return fmt.Errorf("invalid resource type %s, only node types are supported", strings.ToLower(cfg.Operands.Resources[0]))
	}
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	builder := factory.NewBuilder().WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).ContinueOnError().NamespaceParam(namespace).DefaultNamespace()
	if cfg.Selector != "" {
		builder.LabelSelectorParam(cfg.Selector).ResourceTypes("node")
	}
	if cfg.All {
		builder.SelectAllParam(true).ResourceTypes("node").Flatten().Latest()
	}
	if !cfg.All && cfg.Selector == "" && len(cfg.Operands.Resources) >= 2 {
		builder.ResourceNames("node", cfg.Operands.Resources[1:]...)
	}
	builder.LabelSelectorParam(cfg.Selector).Flatten().Latest()
	result := builder.Do()
	if err := result.Err(); err != nil {
		return err
	}

	return result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		oldData, err := json.Marshal(info.Object)
		if err != nil {
			return err
		}
		operation, err := updateNodeTaints(info.Object, cfg.Overwrite, cfg.Operands.Add, cfg.Operands.Remove)
		if err != nil {
			return err
		}
		newData, err := json.Marshal(info.Object)
		if err != nil {
			return err
		}
		patch, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, info.Object)
		createdPatch := patchErr == nil
		if patchErr != nil {
			klog.V(2).Infof("couldn't compute patch: %v", patchErr)
		}

		printFlags := genericclioptions.NewPrintFlags(operation).WithTypeSetter(scheme.Scheme)
		configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
		printObj, err := printObject(printFlags, strategy, cfg.Streams.Out)
		if err != nil {
			return err
		}
		if strategy == util.DryRunClient {
			if createdPatch {
				typed, err := scheme.Scheme.ConvertToVersion(info.Object, info.Mapping.GroupVersionKind.GroupVersion())
				if err != nil {
					return err
				}
				node, ok := typed.(*v1.Node)
				if !ok {
					return fmt.Errorf("unexpected type %T", typed)
				}
				original, err := json.Marshal(node)
				if err != nil {
					return err
				}
				patched, err := strategicpatch.StrategicMergePatch(original, patch, node)
				if err != nil {
					return err
				}
				target, err := runtime.Decode(unstructured.UnstructuredJSONScheme, patched)
				if err != nil {
					return err
				}
				return printObj(target)
			}
			return printObj(info.Object)
		}

		mapping := info.ResourceMapping()
		client, err := factory.ClientForMapping(mapping)
		if err != nil {
			return err
		}
		helper := resource.NewHelper(client, mapping).DryRun(strategy == util.DryRunServer).WithFieldManager(cfg.FieldManager).WithFieldValidation(validationDirective)
		var output runtime.Object
		if createdPatch {
			output, err = helper.Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch, nil)
		} else {
			output, err = helper.Replace(info.Namespace, info.Name, false, info.Object)
		}
		if err != nil {
			return err
		}
		return printObj(output)
	})
}

func updateNodeTaints(object runtime.Object, overwrite bool, add, remove []v1.Taint) (string, error) {
	node, ok := object.(*v1.Node)
	if !ok {
		return "", fmt.Errorf("unexpected type %T, expected Node", object)
	}
	if !overwrite {
		var existing []string
		for _, taint := range add {
			for _, current := range node.Spec.Taints {
				if taint.Key == current.Key && taint.Effect == current.Effect {
					existing = append(existing, taint.Key)
				}
			}
		}
		if len(existing) > 0 {
			return "", fmt.Errorf("node %s already has %v taint(s) with same effect(s) and --overwrite is false", node.Name, strings.Join(existing, ","))
		}
	}
	newTaints := slices.Clone(add)
	for _, current := range node.Spec.Taints {
		found := false
		for _, next := range newTaints {
			if next.MatchTaint(&current) {
				found = true
				break
			}
		}
		if !found {
			newTaints = append(newTaints, current)
		}
	}
	added := len(node.Spec.Taints) != len(newTaints)
	var deleteErrs []error
	deleted := false
	for _, target := range remove {
		before := len(newTaints)
		filtered := newTaints[:0]
		for _, current := range newTaints {
			match := target.Key == current.Key && (target.Effect == "" || target.MatchTaint(&current))
			if !match {
				filtered = append(filtered, current)
			}
		}
		newTaints = filtered
		if len(newTaints) == before {
			deleteErrs = append(deleteErrs, fmt.Errorf("taint %q not found", target.ToString()))
		} else {
			deleted = true
		}
	}
	node.Spec.Taints = newTaints
	operation := "untainted"
	if (added && deleted) || overwrite {
		operation = "modified"
	} else if added {
		operation = "tainted"
	}
	return operation, errors.NewAggregate(deleteErrs)
}

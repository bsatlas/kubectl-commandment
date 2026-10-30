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
	"os"

	"gopkg.in/evanphx/json-patch.v4"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

func runPatch(_ context.Context, cfg *patchConfig) error {
	strategy := cfg.DryRun
	patchType := cfg.Type

	patchBytes := []byte(cfg.Patch)
	if cfg.PatchFile != "" {
		contents, err := os.ReadFile(cfg.PatchFile)
		if err != nil {
			return fmt.Errorf("unable to read patch file: %v", err)
		}
		patchBytes = contents
	}
	patchBytes, err := yaml.ToJSON(patchBytes)
	if err != nil {
		return fmt.Errorf("unable to parse %q: %v", cfg.Patch, err)
	}

	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil && !(cfg.Local && clientcmd.IsEmptyConfig(err)) {
		return err
	}
	filenames := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	result := factory.NewBuilder().
		Unstructured().
		ContinueOnError().
		LocalParam(cfg.Local).
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filenames).
		Subresource(cfg.Subresource).
		ResourceTypeOrNameArgs(false, cfg.Arguments...).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("patched").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	util.PrintFlagsWithDryRunStrategy(printFlags, strategy)
	toPrinter := func(operation string) (func(runtime.Object) error, error) {
		printFlags.NamePrintFlags.Operation = operation
		printer, err := printFlags.ToPrinter()
		if err != nil {
			return nil, err
		}
		return func(object runtime.Object) error { return printer.PrintObj(object, cfg.Streams.Out) }, nil
	}
	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if err := recordFlags.CompleteWithChangeCause("kubectl patch"); err != nil {
		return err
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	count := 0
	err = result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		count++
		if !cfg.Local && strategy != util.DryRunClient {
			mapping := info.ResourceMapping()
			client, err := factory.UnstructuredClientForMapping(mapping)
			if err != nil {
				return err
			}
			helper := resource.NewHelper(client, mapping).
				DryRun(strategy == util.DryRunServer).
				WithFieldManager(cfg.FieldManager).
				WithSubresource(cfg.Subresource)
			patched, err := helper.Patch(info.Namespace, info.Name, patchType, patchBytes, nil)
			if err != nil {
				if errors.IsUnsupportedMediaType(err) {
					return fmt.Errorf("%s is not supported by %s: %w", patchType, mapping.GroupVersionKind, err)
				}
				return err
			}
			didPatch := !equality.Semantic.DeepEqual(info.Object, patched)
			if mergePatch, err := recorder.MakeRecordMergePatch(patched); err == nil && len(mergePatch) != 0 {
				if recorded, err := helper.Patch(info.Namespace, info.Name, types.MergePatchType, mergePatch, nil); err == nil {
					patched = recorded
				}
			}
			printObject, err := toPrinter(patchStatus(didPatch))
			if err != nil {
				return err
			}
			return printObject(patched)
		}

		original, err := runtime.Encode(unstructured.UnstructuredJSONScheme, info.Object)
		if err != nil {
			return err
		}
		patchedJSON, err := patchJSON(patchType, original, patchBytes, info.Object.GetObjectKind().GroupVersionKind())
		if err != nil {
			return err
		}
		patched, err := runtime.Decode(unstructured.UnstructuredJSONScheme, patchedJSON)
		if err != nil {
			return err
		}
		printObject, err := toPrinter(patchStatus(!equality.Semantic.DeepEqual(info.Object, patched)))
		if err != nil {
			return err
		}
		return printObject(patched)
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no objects passed to patch")
	}
	return nil
}

func patchJSON(patchType types.PatchType, original, patch []byte, gvk schema.GroupVersionKind) ([]byte, error) {
	switch patchType {
	case types.JSONPatchType:
		decoded, err := jsonpatch.DecodePatch(patch)
		if err != nil {
			return nil, err
		}
		return decoded.Apply(original)
	case types.MergePatchType:
		return jsonpatch.MergePatch(original, patch)
	case types.StrategicMergePatchType:
		object, err := scheme.Scheme.New(gvk)
		if err != nil {
			return nil, fmt.Errorf("strategic merge patch is not supported for %s locally, try --type merge", gvk.String())
		}
		return strategicpatch.StrategicMergePatch(original, patch, object)
	default:
		return nil, fmt.Errorf("unknown patch type %q", patchType)
	}
}

func patchStatus(changed bool) string {
	if changed {
		return "patched"
	}
	return "patched (no change)"
}

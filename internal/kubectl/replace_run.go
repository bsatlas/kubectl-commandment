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

package kubectl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/rawhttp"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/validation"
)

type forceReplaceOperation struct {
	config           *replaceConfig
	deleteOptions    *delete.DeleteOptions
	schema           validation.Schema
	namespace        string
	enforceNamespace bool
	printObject      func(runtime.Object) error
}

func runReplace(ctx context.Context, cfg *replaceConfig) error {
	factory := cfg.Client.factory()
	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return err
	}

	deleteFlags := delete.NewDeleteFlags("The files, directories or URLs that contain the configurations to replace.")
	*deleteFlags.FileNameFlags.Filenames = slices.Clone(cfg.Filename)
	*deleteFlags.FileNameFlags.Kustomize = cfg.Kustomize
	*deleteFlags.FileNameFlags.Recursive = cfg.Recursive
	*deleteFlags.CascadingStrategy = cfg.Cascade.flagValue()
	*deleteFlags.Force = cfg.Force
	*deleteFlags.GracePeriod = cfg.GracePeriod
	*deleteFlags.Timeout = cfg.Timeout
	*deleteFlags.Wait = cfg.WaitForDeletion
	deleteOptions, err := deleteFlags.ToOptions(dynamicClient, cfg.Streams)
	if err != nil {
		return err
	}
	deleteOptions.IgnoreNotFound = true
	deleteOptions.Output = cfg.Output
	if warning := cfg.Cascade.deprecationWarning(); warning != "" {
		fmt.Fprintln(deleteOptions.ErrOut, warning)
	}

	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	if cfg.Raw != "" {
		filename := deleteOptions.FilenameOptions.Filenames[0]
		client, err := factory.RESTClient()
		if err != nil {
			return err
		}
		return rawhttp.RawPut(client, cfg.Streams, cfg.Raw, filename)
	}

	schema, err := factory.Validator(cfg.Validate)
	if err != nil {
		return err
	}
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("replaced").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	util.PrintFlagsWithDryRunStrategy(printFlags, strategy)
	printer, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}
	printObject := func(object runtime.Object) error {
		return printer.PrintObj(object, cfg.Streams.Out)
	}

	if deleteOptions.ForceDeletion {
		operation := forceReplaceOperation{
			config:           cfg,
			deleteOptions:    deleteOptions,
			schema:           schema,
			namespace:        namespace,
			enforceNamespace: enforceNamespace,
			printObject:      printObject,
		}
		return operation.run(ctx)
	}

	recorder := genericclioptions.NoopRecorder{}

	result := factory.NewBuilder().
		Unstructured().
		Schema(schema).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &deleteOptions.FilenameOptions).
		Subresource(cfg.Subresource).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	return result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if err := saveConfigAnnotation(cfg.SaveConfig, info.Object); err != nil {
			return util.AddSourceToErr("replacing", info.Source, err)
		}
		if err := recorder.Record(info.Object); err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		}
		if strategy == util.DryRunClient {
			return printObject(info.Object)
		}
		object, err := resource.NewHelper(info.Client, info.Mapping).
			DryRun(strategy == util.DryRunServer).
			WithFieldManager(cfg.FieldManager).
			WithFieldValidation(cfg.Validate).
			WithSubresource(cfg.Subresource).
			Replace(info.Namespace, info.Name, true, info.Object)
		if err != nil {
			return util.AddSourceToErr("replacing", info.Source, err)
		}
		info.Refresh(object, true)
		return printObject(info.Object)
	})
}

func (operation forceReplaceOperation) run(ctx context.Context) error {
	cfg := operation.config
	deleteOptions := operation.deleteOptions
	stdinInUse := false
	for i, filename := range deleteOptions.FilenameOptions.Filenames {
		if filename != "-" {
			continue
		}
		tempDir, err := os.MkdirTemp("", "kubectl_replace_")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)
		tempFilename := filepath.Join(tempDir, "resource.stdin")
		if err := util.DumpReaderToFile(cfg.Streams.In, tempFilename); err != nil {
			return err
		}
		deleteOptions.FilenameOptions.Filenames[i] = tempFilename
		stdinInUse = true
	}

	factory := cfg.Client.factory()
	builder := factory.NewBuilder().
		Unstructured().
		ContinueOnError().
		NamespaceParam(operation.namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(false, cfg.Arguments...).RequireObject(false).
		FilenameParam(operation.enforceNamespace, &deleteOptions.FilenameOptions).
		Subresource(cfg.Subresource).
		Flatten()
	if stdinInUse {
		builder = builder.StdinInUse()
	}
	result := builder.Do()
	if err := result.Err(); err != nil {
		return err
	}
	if err := deleteOptions.DeleteResult(result); err != nil {
		return err
	}

	timeout := deleteOptions.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	if err := result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		return wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(_ context.Context) (bool, error) {
			err := info.Get()
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
	}); err != nil {
		return err
	}

	builder = factory.NewBuilder().
		Unstructured().
		Schema(operation.schema).
		ContinueOnError().
		NamespaceParam(operation.namespace).DefaultNamespace().
		FilenameParam(operation.enforceNamespace, &deleteOptions.FilenameOptions).
		Subresource(cfg.Subresource).
		Flatten()
	if stdinInUse {
		builder = builder.StdinInUse()
	}
	result = builder.Do()
	if err := result.Err(); err != nil {
		return err
	}

	count := 0
	if err := result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if err := saveConfigAnnotation(cfg.SaveConfig, info.Object); err != nil {
			return err
		}
		recorder := genericclioptions.NoopRecorder{}
		if err := recorder.Record(info.Object); err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		}
		object, err := resource.NewHelper(info.Client, info.Mapping).
			WithFieldManager(cfg.FieldManager).
			WithFieldValidation(cfg.Validate).
			Create(info.Namespace, true, info.Object)
		if err != nil {
			return err
		}
		count++
		info.Refresh(object, true)
		return operation.printObject(info.Object)
	}); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no objects passed to replace")
	}
	return nil
}

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
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/get"
	"k8s.io/kubectl/pkg/scheme"
)

func runGet(_ context.Context, cfg *getConfig) error {
	factory := cfg.Client.factory()
	streams := genericiooptions.IOStreams{
		In:     cfg.Stdin,
		Out:    cfg.Stdout,
		ErrOut: cfg.Stderr,
	}
	options := get.GetOptions{
		PrintFlags:             getPrintFlags(cfg),
		IsHumanReadablePrinter: cfg.HumanReadable,
		FilenameOptions: resource.FilenameOptions{
			Filenames: cfg.Filenames,
			Kustomize: cfg.Kustomize,
			Recursive: cfg.Recursive,
		},
		Raw:               cfg.Raw,
		Watch:             cfg.Watch,
		WatchOnly:         cfg.WatchOnly,
		ChunkSize:         cfg.ChunkSize,
		OutputWatchEvents: cfg.OutputWatchEvents,
		LabelSelector:     cfg.LabelSelector,
		FieldSelector:     cfg.FieldSelector,
		AllNamespaces:     cfg.AllNamespaces,
		Subresource:       cfg.Subresource,
		SortBy:            cfg.SortBy,
		ServerPrint:       cfg.ServerPrint,
		NoHeaders:         cfg.NoHeaders,
		IgnoreNotFound:    cfg.IgnoreNotFound,
		IOStreams:         streams,
	}

	if cfg.Raw == "" {
		namespace, explicitNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}
		if cfg.AllNamespaces {
			explicitNamespace = false
		}
		options.Namespace = namespace
		options.ExplicitNamespace = explicitNamespace
	}

	options.ToPrinter = func(mapping *meta.RESTMapping, outputObjects *bool, withNamespace, withKind bool) (printers.ResourcePrinterFunc, error) {
		flags := options.PrintFlags.Copy()
		if mapping != nil {
			flags.SetKind(mapping.GroupVersionKind.GroupKind())
		}
		if withNamespace {
			_ = flags.EnsureWithNamespace()
		}
		if withKind {
			_ = flags.EnsureWithKind()
		}

		printer, err := flags.ToPrinter()
		if err != nil {
			return nil, err
		}
		printer, err = printers.NewTypeSetter(scheme.Scheme).WrapToPrinter(printer, nil)
		if err != nil {
			return nil, err
		}
		if cfg.SortBy != "" {
			printer = &get.SortingPrinter{
				Delegate:  printer,
				SortField: cfg.SortBy,
			}
		}
		if outputObjects != nil {
			delegate := printer
			printer = printers.ResourcePrinterFunc(func(object runtime.Object, writer io.Writer) error {
				if !*outputObjects {
					return nil
				}
				return delegate.PrintObj(object, writer)
			})
		}
		if cfg.ServerPrint {
			printer = &get.TablePrinter{
				Delegate: printer,
			}
		}
		return printer.PrintObj, nil
	}

	if cfg.SortBy != "" {
		switch {
		case cfg.Watch:
			_, _ = fmt.Fprintln(cfg.Stderr, "warning: --watch requested, --sort-by will be ignored for watch events received")
		case cfg.WatchOnly:
			_, _ = fmt.Fprintln(cfg.Stderr, "warning: --watch-only requested, --sort-by will be ignored")
		}
	}

	resourceArguments := append([]string(nil), cfg.QualifiedResources...)
	if cfg.ResourceType != "" {
		resourceArguments = make([]string, 0, 1+len(cfg.Names))
		resourceArguments = append(resourceArguments, cfg.ResourceType)
		resourceArguments = append(resourceArguments, cfg.Names...)
	}
	return options.Run(factory, resourceArguments)
}

func getPrintFlags(cfg *getConfig) *get.PrintFlags {
	flags := get.NewGetPrintFlags()
	*flags.OutputFormat = cfg.Output
	*flags.NoHeaders = cfg.NoHeaders
	flags.JSONYamlPrintFlags.ShowManagedFields = cfg.ShowManagedFields
	*flags.TemplateFlags.TemplateArgument = cfg.Template
	*flags.TemplateFlags.AllowMissingKeys = cfg.AllowMissingTemplateKeys
	*flags.HumanReadableFlags.ShowLabels = cfg.ShowLabels
	*flags.HumanReadableFlags.SortBy = cfg.SortBy
	*flags.HumanReadableFlags.ColumnLabels = append([]string(nil), cfg.LabelColumns...)
	*flags.HumanReadableFlags.ShowKind = cfg.ShowKind
	return flags
}

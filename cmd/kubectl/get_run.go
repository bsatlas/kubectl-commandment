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
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/get"
	"k8s.io/kubectl/pkg/rawhttp"
	"k8s.io/kubectl/pkg/scheme"
)

func runGet(ctx context.Context, cfg *getConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	getter := restClientGetter(cfg.Client)
	if cfg.Raw != "" {
		return runRaw(getter, cfg)
	}
	if cfg.Watch || cfg.WatchOnly {
		return watchGet(ctx, getter, cfg)
	}

	namespace, explicitNamespace, err := getter.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	if cfg.AllNamespaces {
		explicitNamespace = false
	}

	chunkSize := cfg.ChunkSize
	if cfg.SortBy != "" {
		chunkSize = 0
	}
	humanReadable := (cfg.Output == "" && cfg.Template == "") || cfg.Output == "wide"
	serverPrint := cfg.ServerPrint
	if strings.Contains(cfg.Output, "custom-columns") || cfg.Output == "yaml" || strings.Contains(cfg.Output, "json") {
		serverPrint = false
	}

	filenameOptions := &resource.FilenameOptions{
		Filenames: cfg.Filenames,
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	result := resource.NewBuilder(getter).
		Unstructured().
		NamespaceParam(namespace).
		DefaultNamespace().
		AllNamespaces(cfg.AllNamespaces).
		FilenameParam(explicitNamespace, filenameOptions).
		LabelSelectorParam(cfg.LabelSelector).
		FieldSelectorParam(cfg.FieldSelector).
		Subresource(cfg.Subresource).
		RequestChunksOf(chunkSize).
		ResourceTypeOrNameArgs(true, resourceArguments(cfg)...).
		ContinueOnError().
		Latest().
		Flatten().
		TransformRequests(transformRequests(serverPrint, humanReadable, cfg.SortBy)).
		Do()

	if cfg.IgnoreNotFound {
		result.IgnoreErrors(errors.IsNotFound)
	}
	if err := result.Err(); err != nil {
		return err
	}
	if humanReadable {
		return printHuman(result, cfg, namespace, serverPrint)
	}
	return printGeneric(result, cfg)
}

func watchGet(ctx context.Context, getter genericclioptions.RESTClientGetter, cfg *getConfig) error {
	namespace, explicitNamespace, err := getter.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	if cfg.AllNamespaces {
		explicitNamespace = false
	}
	if cfg.SortBy != "" {
		if cfg.WatchOnly {
			_, _ = fmt.Fprintln(cfg.Stderr, "warning: --watch-only requested, --sort-by will be ignored")
		} else {
			_, _ = fmt.Fprintln(cfg.Stderr, "warning: --watch requested, --sort-by will be ignored for watch events received")
		}
	}

	result := resource.NewBuilder(getter).
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().
		AllNamespaces(cfg.AllNamespaces).
		FilenameParam(explicitNamespace, &resource.FilenameOptions{
			Filenames: cfg.Filenames,
			Kustomize: cfg.Kustomize,
			Recursive: cfg.Recursive,
		}).
		LabelSelectorParam(cfg.LabelSelector).
		FieldSelectorParam(cfg.FieldSelector).
		RequestChunksOf(cfg.ChunkSize).
		ResourceTypeOrNameArgs(true, resourceArguments(cfg)...).
		SingleResourceType().
		Latest().
		TransformRequests(transformRequests(cfg.ServerPrint, true, "")).
		Do()
	if err := result.Err(); err != nil {
		return err
	}
	infos, err := result.Infos()
	if err != nil {
		if cfg.IgnoreNotFound && errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if len(infos) == 0 {
		if cfg.IgnoreNotFound {
			return nil
		}
		return fmt.Errorf("no resources found to watch")
	}

	info := infos[0]
	outputObjects := !cfg.WatchOnly
	printer, err := watchPrinter(cfg, info.Mapping, &outputObjects)
	if err != nil {
		return err
	}
	object, err := result.Object()
	if err != nil {
		return err
	}

	resourceVersion := "0"
	isList := meta.IsListType(object)
	if isList {
		resourceVersion, err = meta.NewAccessor().ResourceVersion(object)
		if err != nil {
			return err
		}
	}

	writer := printers.GetNewTabWriter(cfg.Stdout)
	var current []runtime.Object
	if isList {
		current, err = meta.ExtractList(object)
		if err != nil {
			return err
		}
	} else {
		current = []runtime.Object{object}
	}
	for _, item := range current {
		if cfg.OutputWatchEvents {
			item = &v1.WatchEvent{
				Type: string(watch.Added),
				Object: runtime.RawExtension{
					Object: item,
				},
			}
		}
		if err := printer.PrintObj(item, writer); err != nil {
			return fmt.Errorf("unable to output the provided object: %v", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	outputObjects = isList

	stream, err := result.Watch(resourceVersion)
	if err != nil {
		return err
	}
	defer stream.Stop()
	err = runGetWatchUntil(ctx, stream, func(event watch.Event) (bool, error) {
		item := event.Object
		if cfg.OutputWatchEvents {
			item = &v1.WatchEvent{
				Type: string(event.Type),
				Object: runtime.RawExtension{
					Object: item,
				},
			}
		}
		if err := printer.PrintObj(item, writer); err != nil {
			return false, err
		}
		if err := writer.Flush(); err != nil {
			return false, err
		}
		outputObjects = true
		return false, nil
	})
	return err
}

func watchPrinter(cfg *getConfig, mapping *meta.RESTMapping, output *bool) (printers.ResourcePrinter, error) {
	flags := printFlags(cfg)
	if mapping != nil {
		flags.SetKind(mapping.GroupVersionKind.GroupKind())
		if cfg.AllNamespaces && mapping.Scope.Name() != meta.RESTScopeNameRoot {
			if err := flags.EnsureWithNamespace(); err != nil {
				return nil, err
			}
		}
	}
	printer, err := flags.ToPrinter()
	if err != nil {
		return nil, err
	}
	printer, err = printers.NewTypeSetter(scheme.Scheme).WrapToPrinter(printer, nil)
	if err != nil {
		return nil, err
	}
	printer = &watchSkipPrinter{
		delegate: printer,
		output:   output,
	}
	if cfg.ServerPrint {
		printer = &get.TablePrinter{
			Delegate: printer,
		}
	}
	return printer, nil
}

type watchSkipPrinter struct {
	delegate printers.ResourcePrinter
	output   *bool
}

func (printer *watchSkipPrinter) PrintObj(object runtime.Object, writer io.Writer) error {
	if printer.output == nil || *printer.output {
		return printer.delegate.PrintObj(object, writer)
	}
	return nil
}

func resourceArguments(cfg *getConfig) []string {
	if cfg.ResourceType != "" {
		arguments := make([]string, 0, 1+len(cfg.Names))
		arguments = append(arguments, cfg.ResourceType)
		arguments = append(arguments, cfg.Names...)
		return arguments
	}
	return append([]string(nil), cfg.QualifiedResources...)
}

func runRaw(getter resource.RESTClientGetter, cfg *getConfig) error {
	clientConfig, err := getter.ToRESTConfig()
	if err != nil {
		return err
	}
	clientConfig = rest.CopyConfig(clientConfig)
	clientConfig.GroupVersion = &schema.GroupVersion{
		Version: "v1",
	}
	if clientConfig.APIPath == "" {
		clientConfig.APIPath = "/api"
	}
	if clientConfig.NegotiatedSerializer == nil {
		clientConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	}
	if err := rest.SetKubernetesDefaults(clientConfig); err != nil {
		return err
	}
	restClient, err := rest.RESTClientFor(clientConfig)
	if err != nil {
		return err
	}
	streams := genericiooptions.IOStreams{
		In:     cfg.Stdin,
		Out:    cfg.Stdout,
		ErrOut: cfg.Stderr,
	}
	return rawhttp.RawGet(restClient, streams, cfg.Raw)
}

func transformRequests(serverPrint, humanReadable bool, sortBy string) func(*rest.Request) {
	return func(request *rest.Request) {
		if !serverPrint || !humanReadable {
			return
		}
		request.SetHeader("Accept", strings.Join([]string{
			getTableV1MediaType(),
			getTableV1Beta1MediaType(),
			"application/json",
		}, ","))
		if sortBy != "" {
			request.Param("includeObject", "Object")
		}
	}
}

func printFlags(cfg *getConfig) *get.PrintFlags {
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

func printGeneric(result *resource.Result, cfg *getConfig) error {
	var singleItemImplied bool
	infos, err := result.IntoSingleItemImplied(&singleItemImplied).Infos()
	if err != nil {
		if singleItemImplied {
			return err
		}
		if len(infos) == 0 {
			return err
		}
	}
	if len(infos) == 0 && cfg.IgnoreNotFound {
		return nil
	}

	printer, err := printFlags(cfg).ToPrinter()
	if err != nil {
		return err
	}
	printer, err = printers.NewTypeSetter(scheme.Scheme).WrapToPrinter(printer, nil)
	if err != nil {
		return err
	}
	if cfg.SortBy != "" {
		printer = &get.SortingPrinter{
			Delegate:  printer,
			SortField: cfg.SortBy,
		}
	}

	var object runtime.Object
	if singleItemImplied && len(infos) == 1 {
		object = infos[0].Object
	} else {
		list := &unstructured.UnstructuredList{}
		list.SetAPIVersion("v1")
		list.SetKind("List")
		for _, info := range infos {
			item, ok := info.Object.(*unstructured.Unstructured)
			if !ok {
				return fmt.Errorf("unexpected object type %T in generic get result", info.Object)
			}
			list.Items = append(list.Items, *item.DeepCopy())
		}
		object = list
	}
	return printer.PrintObj(object, cfg.Stdout)
}

func printHuman(result *resource.Result, cfg *getConfig, namespace string, serverPrint bool) error {
	infos, err := result.Infos()
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		if cfg.IgnoreNotFound {
			return nil
		}
		if cfg.AllNamespaces {
			_, err = fmt.Fprintln(cfg.Stderr, "No resources found")
		} else {
			_, err = fmt.Fprintf(cfg.Stderr, "No resources found in %s namespace.\n", namespace)
		}
		return err
	}

	printWithKind := multipleGroupVersionKinds(infos)
	writer := printers.GetNewTabWriter(cfg.Stdout)
	for _, info := range infos {
		flags := printFlags(cfg)
		if info.Mapping != nil {
			flags.SetKind(info.Mapping.GroupVersionKind.GroupKind())
			if cfg.AllNamespaces && info.Mapping.Scope.Name() != meta.RESTScopeNameRoot {
				if err := flags.EnsureWithNamespace(); err != nil {
					return err
				}
			}
		}
		if printWithKind {
			if err := flags.EnsureWithKind(); err != nil {
				return err
			}
		}
		printer, err := flags.ToPrinter()
		if err != nil {
			return err
		}
		printer, err = printers.NewTypeSetter(scheme.Scheme).WrapToPrinter(printer, nil)
		if err != nil {
			return err
		}
		if serverPrint {
			printer = &get.TablePrinter{
				Delegate: printer,
			}
		}
		if err := printer.PrintObj(info.Object, writer); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func multipleGroupVersionKinds(infos []*resource.Info) bool {
	if len(infos) < 2 || infos[0].Mapping == nil {
		return false
	}
	first := infos[0].Mapping.GroupVersionKind
	for _, info := range infos[1:] {
		if info.Mapping == nil || info.Mapping.GroupVersionKind != first {
			return true
		}
	}
	return false
}

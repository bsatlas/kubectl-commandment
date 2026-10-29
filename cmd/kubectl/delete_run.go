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
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/delete"
)

func runDelete(_ context.Context, cfg *deleteConfig) error {
	factory := cfg.Client.factory()
	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return err
	}
	flags := delete.NewDeleteCommandFlags("containing the resource to delete.")
	*flags.FileNameFlags.Filenames = slices.Clone(cfg.Filename)
	*flags.FileNameFlags.Kustomize = cfg.Kustomize
	*flags.FileNameFlags.Recursive = cfg.Recursive
	*flags.LabelSelector = cfg.Selector
	*flags.FieldSelector = cfg.FieldSelector
	*flags.All = cfg.All
	*flags.AllNamespaces = cfg.AllNamespaces
	*flags.CascadingStrategy = cfg.Cascade.flagValue()
	*flags.Force = cfg.Force
	*flags.GracePeriod = cfg.GracePeriod
	*flags.IgnoreNotFound = cfg.IgnoreNotFound
	*flags.Now = cfg.Now
	*flags.Timeout = cfg.Timeout
	*flags.Wait = cfg.Wait
	*flags.Output = cfg.Output
	*flags.Raw = cfg.Raw
	*flags.Interactive = cfg.Interactive
	options, err := flags.ToOptions(dynamicClient, cfg.Streams)
	if err != nil {
		return err
	}

	if warning := cfg.Cascade.deprecationWarning(); warning != "" {
		fmt.Fprintln(options.ErrOut, warning)
	}

	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.DryRunStrategy = strategy
	warningOptions := printers.WarningPrinterOptions{
		Color: printers.AllowsColorOutput(options.ErrOut),
	}
	options.WarningPrinter = printers.NewWarningPrinter(options.ErrOut, warningOptions)

	if options.Raw == "" {
		namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}
		options.WarnClusterScope = enforceNamespace && !options.DeleteAllNamespaces
		build := func() *resource.Result {
			return factory.NewBuilder().
				Unstructured().
				ContinueOnError().
				NamespaceParam(namespace).DefaultNamespace().
				FilenameParam(enforceNamespace, &options.FilenameOptions).
				LabelSelectorParam(options.LabelSelector).
				FieldSelectorParam(options.FieldSelector).
				SelectAllParam(options.DeleteAll).
				AllNamespaces(options.DeleteAllNamespaces).
				ResourceTypeOrNameArgs(false, cfg.Arguments...).RequireObject(false).
				Flatten().
				Do()
		}
		options.Result = build()
		if err := options.Result.Err(); err != nil {
			return err
		}
		if options.Interactive {
			preview := build()
			if err := preview.Err(); err != nil {
				return err
			}
			var infos []*resource.Info
			if options.IgnoreNotFound {
				preview = preview.IgnoreErrors(func(err error) bool {
					return errors.IsNotFound(err)
				})
			}
			if err := preview.Visit(func(info *resource.Info, err error) error {
				if err != nil {
					return err
				}
				infos = append(infos, info)
				return nil
			}); err != nil {
				return err
			}
			if len(infos) == 0 {
				fmt.Fprintln(options.Out, "No resources found")
				return nil
			}
			if !confirmDelete(options.IOStreams, infos) {
				fmt.Fprintln(options.Out, "deletion is cancelled")
				return nil
			}
			options.Interactive = false
		}
	}

	if options.GracePeriod == 0 && options.ForceDeletion {
		options.WarningPrinter.Print("Immediate deletion does not wait for confirmation that the running resource has been terminated. The resource may continue to run on the cluster indefinitely.")
	}
	return options.RunDelete(factory)
}

func confirmDelete(streams genericiooptions.IOStreams, infos []*resource.Info) bool {
	fmt.Fprintf(streams.Out, "You are about to delete the following %d resource(s):\n", len(infos))
	for _, info := range infos {
		kind := strings.ToLower(info.Mapping.GroupVersionKind.Kind)
		if group := info.Mapping.GroupVersionKind.Group; group != "" {
			kind += "." + group
		}
		fmt.Fprintf(streams.Out, "%s/%s\n", kind, info.Name)
	}
	fmt.Fprint(streams.Out, "Do you want to continue? (y/N): ")
	var input string
	if _, err := fmt.Fscanln(streams.In, &input); err != nil {
		return false
	}
	return strings.EqualFold(input, "y")
}

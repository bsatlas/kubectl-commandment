package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/yaml"
)

func runApplyViewLastApplied(_ context.Context, cfg *applyViewLastAppliedConfig) error {
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	filenameOptions := resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filename),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	result := factory.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filenameOptions).
		ResourceTypeOrNameArgs(enforceNamespace, cfg.Arguments...).
		SelectAllParam(cfg.All).
		LabelSelectorParam(cfg.Selector).
		Latest().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}
	return result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		configuration, err := util.GetOriginalConfiguration(info.Object)
		if err != nil {
			return err
		}
		if configuration == nil {
			return fmt.Errorf("no last-applied-configuration annotation found on resource: %s", info.Name)
		}
		if cfg.Output == "json" {
			buffer := &bytes.Buffer{}
			if err := json.Indent(buffer, configuration, "", "  "); err != nil {
				return err
			}
			_, err = fmt.Fprint(cfg.Streams.Out, buffer.String())
			return err
		}
		output, err := yaml.JSONToYAML(configuration)
		if err != nil {
			return err
		}
		_, err = cfg.Streams.Out.Write(output)
		return err
	})
}

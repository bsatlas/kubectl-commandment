package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/yaml"
)

func runApplyViewLastApplied(ctx context.Context, cfg *applyViewLastAppliedConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	filenameOptions := resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	builder := factory.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace()
	addFilenameSources(builder, enforceNamespace, filenameOptions, cfg.Streams.In)
	result := builder.
		ResourceTypeOrNameArgs(enforceNamespace, cfg.Arguments...).
		SelectAllParam(cfg.All).
		LabelSelectorParam(cfg.Selector).
		Latest().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	var configurations [][]byte
	if err := result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		configuration, err := util.GetOriginalConfiguration(info.Object)
		if err != nil {
			return err
		}
		if configuration == nil {
			return cmdutil.AddSourceToErr(fmt.Sprintf("no last-applied-configuration annotation found on resource: %s\n", info.Name), info.Source, err)
		}
		configurations = append(configurations, configuration)
		return nil
	}); err != nil {
		return err
	}

	for _, configuration := range configurations {
		if cfg.Output == "json" {
			buffer := &bytes.Buffer{}
			if err := json.Indent(buffer, configuration, "", "  "); err != nil {
				return err
			}
			if _, err := fmt.Fprint(cfg.Streams.Out, buffer.String()); err != nil {
				return err
			}
			continue
		}
		output, err := yaml.JSONToYAML(configuration)
		if err != nil {
			return err
		}
		if _, err := cfg.Streams.Out.Write(output); err != nil {
			return err
		}
	}
	return nil
}

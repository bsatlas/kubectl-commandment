package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/yaml"
)

type lastAppliedPatch struct {
	info      *resource.Info
	patch     []byte
	patchType types.PatchType
}

func runApplySetLastApplied(_ context.Context, cfg *applySetLastAppliedConfig) error {
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	filenameOptions := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
	}
	result := factory.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filenameOptions).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	patches := make([]lastAppliedPatch, 0)
	if err := result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		patch, diff, patchType, err := editor.GetApplyPatch(info.Object.(runtime.Unstructured))
		if err != nil {
			return err
		}
		if err := info.Get(); err != nil {
			if errors.IsNotFound(err) {
				return err
			}
			return util.AddSourceToErr(fmt.Sprintf("retrieving current configuration of:\n%s\nfrom server for:", info.String()), info.Source, err)
		}
		original, err := originalConfiguration(info.Object)
		if err != nil {
			return util.AddSourceToErr(fmt.Sprintf("retrieving current configuration of:\n%s\nfrom server for:", info.String()), info.Source, err)
		}
		if original == nil && !cfg.CreateAnnotation {
			return fmt.Errorf("no last-applied-configuration annotation found on resource: %s, to create the annotation, run the command with --create-annotation", info.Name)
		}
		if bytes.Equal(util.StripComments(original), util.StripComments(diff)) {
			fmt.Fprintf(cfg.Streams.Out, "set-last-applied %s: no changes required.\n", info.Name)
			return nil
		}
		patches = append(patches, lastAppliedPatch{
			info:      info,
			patch:     patch,
			patchType: patchType,
		})
		return nil
	}); err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("configured").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printObject, err := printObject(printFlags, strategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	for _, pending := range patches {
		object := pending.info.Object
		if strategy != util.DryRunClient {
			mapping := pending.info.ResourceMapping()
			client, err := factory.UnstructuredClientForMapping(mapping)
			if err != nil {
				return err
			}
			helper := resource.NewHelper(client, mapping).DryRun(strategy == util.DryRunServer)
			object, err = helper.Patch(pending.info.Namespace, pending.info.Name, pending.patchType, pending.patch, nil)
			if err != nil {
				return err
			}
		}
		if err := printObject(object); err != nil {
			return err
		}
	}
	return nil
}

func runApplyViewLastApplied(_ context.Context, cfg *applyViewLastAppliedConfig) error {
	if cfg.Output != "yaml" && cfg.Output != "json" {
		return fmt.Errorf("unexpected -o output mode: %s, the flag 'output' must be one of yaml|json", cfg.Output)
	}
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	filenameOptions := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
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
		configuration, err := originalConfiguration(info.Object)
		if err != nil {
			return err
		}
		if configuration == nil {
			return fmt.Errorf("no last-applied-configuration annotation found on resource: %s", info.Name)
		}
		switch cfg.Output {
		case "json":
			buffer := &bytes.Buffer{}
			if err := json.Indent(buffer, configuration, "", "  "); err != nil {
				return err
			}
			_, err = fmt.Fprint(cfg.Streams.Out, buffer.String())
			return err
		case "yaml":
			output, err := yaml.JSONToYAML(configuration)
			if err != nil {
				return err
			}
			_, err = cfg.Streams.Out.Write(output)
			return err
		default:
			return nil
		}
	})
}

package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"k8s.io/kubectl/pkg/scheme"
)

type lastAppliedPatch struct {
	info      *resource.Info
	patch     []byte
	patchType types.PatchType
}

func runApplySetLastApplied(_ context.Context, cfg *applySetLastAppliedConfig) error {
	strategy := cfg.DryRun
	factory := cfg.Client.factory()
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	filenameOptions := resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filename),
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

	var patches []lastAppliedPatch
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
		pending := lastAppliedPatch{
			info:      info,
			patch:     patch,
			patchType: patchType,
		}
		patches = append(patches, pending)
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

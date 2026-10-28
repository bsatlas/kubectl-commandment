package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/openapi3"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/apply"
	"k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/prune"
)

func runApply(_ context.Context, cfg *applyConfig) error {
	factory := cfg.Client.factory()
	strategy := cfg.DryRun

	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return err
	}
	deleteFlags := delete.NewDeleteFlags("The files, directories or URLs that contain the configurations to apply.")
	*deleteFlags.FileNameFlags.Filenames = append([]string(nil), cfg.Filename...)
	*deleteFlags.FileNameFlags.Kustomize = cfg.Kustomize
	*deleteFlags.FileNameFlags.Recursive = cfg.Recursive
	*deleteFlags.CascadingStrategy = cfg.Cascade
	*deleteFlags.Force = cfg.Force
	*deleteFlags.GracePeriod = cfg.GracePeriod
	*deleteFlags.Timeout = cfg.Timeout
	*deleteFlags.Wait = cfg.Wait
	deleteOptions, err := deleteFlags.ToOptions(dynamicClient, cfg.Streams)
	if err != nil {
		return err
	}
	if err := deleteOptions.FilenameOptions.RequireFilenameOrKustomize(); err != nil {
		return err
	}
	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	toPrinter := func(operation string) (printers.ResourcePrinter, error) {
		printFlags.NamePrintFlags.Operation = operation
		util.PrintFlagsWithDryRunStrategy(printFlags, strategy)
		return printFlags.ToPrinter()
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl apply"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	validate, err := validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	validation, err := factory.Validator(validate)
	if err != nil {
		return err
	}
	mapper, err := factory.ToRESTMapper()
	if err != nil {
		return err
	}
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	openAPIV3Client, openAPIV3Err := factory.OpenAPIV3Client()
	var openAPIV3Root openapi3.Root
	if openAPIV3Err == nil {
		openAPIV3Root = openapi3.NewRoot(openAPIV3Client)
	} else {
		klog.V(4).Infof("warning: OpenAPI V3 Patch is enabled but is unable to be loaded. Will fall back to OpenAPI V2")
	}

	var applySet *apply.ApplySet
	if cfg.ApplySet != "" {
		parent, err := apply.ParseApplySetParentRef(cfg.ApplySet, mapper)
		if err != nil {
			return fmt.Errorf("invalid parent reference %q: %w", cfg.ApplySet, err)
		}
		if enforceNamespace && parent.IsNamespaced() {
			parent.Namespace = namespace
		}
		tooling := apply.ApplySetTooling{
			Name:    "kubectl",
			Version: apply.ApplySetToolVersion,
		}
		restClient, err := factory.UnstructuredClientForMapping(parent.RESTMapping)
		if err != nil {
			return fmt.Errorf("failed to initialize RESTClient for ApplySet: %w", err)
		}
		if restClient == nil {
			return fmt.Errorf("could not build RESTClient for ApplySet")
		}
		applySet = apply.NewApplySet(parent, tooling, mapper, restClient)
	}

	pruneResources := []prune.Resource(nil)
	if cfg.Prune {
		pruneResources, err = prune.ParseResources(mapper, cfg.PruneAllowlist)
		if err != nil {
			return err
		}
	}

	options := &apply.ApplyOptions{
		Recorder:            recorder,
		PrintFlags:          printFlags,
		ToPrinter:           toPrinter,
		DeleteOptions:       deleteOptions,
		ServerSideApply:     cfg.ServerSide,
		ForceConflicts:      cfg.ForceConflicts,
		FieldManager:        cfg.FieldManager,
		Selector:            cfg.Selector,
		DryRunStrategy:      strategy,
		Prune:               cfg.Prune,
		PruneResources:      pruneResources,
		All:                 cfg.All,
		Overwrite:           cfg.Overwrite,
		OpenAPIPatch:        cfg.OpenapiPatch,
		Subresource:         cfg.Subresource,
		ValidationDirective: validate,
		Validator:           validation,
		Builder:             factory.NewBuilder(),
		Mapper:              mapper,
		DynamicClient:       dynamicClient,
		OpenAPIGetter:       factory,
		OpenAPIV3Root:       openAPIV3Root,
		Namespace:           namespace,
		EnforceNamespace:    enforceNamespace,
		IOStreams:           cfg.Streams,
		VisitedUids:         sets.New[types.UID](),
		VisitedNamespaces:   sets.New[string](),
		ApplySet:            applySet,
	}
	options.PostProcessorFn = options.PrintAndPrunePostProcessor()
	return options.Run()
}

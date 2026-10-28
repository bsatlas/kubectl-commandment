package main

import (
	"context"

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
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}

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

	validation, err := factory.Validator(cfg.Validate)
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

	pruneResources := []prune.Resource(nil)
	if cfg.Prune {
		pruneResources, err = prune.ParseResources(mapper, cfg.PruneAllowlist)
		if err != nil {
			return err
		}
	}

	fieldManager := cfg.FieldManager
	if cfg.ServerSide && !explicitFlagValue(cfg.FieldManagerState) {
		fieldManager = "kubectl"
	}

	options := &apply.ApplyOptions{
		Recorder:            recorder,
		PrintFlags:          printFlags,
		ToPrinter:           toPrinter,
		DeleteOptions:       deleteOptions,
		ServerSideApply:     cfg.ServerSide,
		ForceConflicts:      cfg.ForceConflicts,
		FieldManager:        fieldManager,
		Selector:            cfg.Selector,
		DryRunStrategy:      strategy,
		Prune:               cfg.Prune,
		PruneResources:      pruneResources,
		All:                 cfg.All,
		Overwrite:           cfg.Overwrite,
		OpenAPIPatch:        cfg.OpenapiPatch,
		Subresource:         cfg.Subresource,
		ValidationDirective: cfg.Validate,
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
	}
	options.PostProcessorFn = options.PrintAndPrunePostProcessor()
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

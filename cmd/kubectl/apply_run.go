package main

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/openapi3"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/apply"
	"k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util/prune"
)

var errApplySetRESTClientUnavailable = errors.New("could not build RESTClient for ApplySet")

func runApply(ctx context.Context, cfg *applyConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	strategy := cfg.DryRun

	dynamicClient, err := factory.DynamicClient()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
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

	deleteFlags := delete.NewDeleteFlags("The files, directories or URLs that contain the configurations to apply.")
	*deleteFlags.FileNameFlags.Filenames = cfg.Filename.Filenames()
	*deleteFlags.FileNameFlags.Kustomize = cfg.Filename.Kustomize
	*deleteFlags.FileNameFlags.Recursive = cfg.Filename.Recursive
	*deleteFlags.CascadingStrategy = cfg.Cascade
	*deleteFlags.Force = cfg.Force
	*deleteFlags.GracePeriod = cfg.GracePeriod
	*deleteFlags.Timeout = cfg.Timeout
	*deleteFlags.Wait = cfg.Wait
	deleteOptions, err := deleteFlags.ToOptions(dynamicClient, cfg.Streams)
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
			return errApplySetRESTClientUnavailable
		}
		applySet = apply.NewApplySet(parent, tooling, mapper, restClient)
		if err := applySet.Validate(ctx, dynamicClient); err != nil {
			return err
		}
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

	addFilenameSources(options.Builder, options.EnforceNamespace, options.DeleteOptions.FilenameOptions, cfg.Streams.In)
	options.DeleteOptions.FilenameOptions = resource.FilenameOptions{}

	return options.Run()
}

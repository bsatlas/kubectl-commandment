package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

var errRolloutHistoryRevisionNotFound = errors.New("unable to find the specified revision")

func runRolloutHistory(ctx context.Context, cfg *rolloutHistoryConfig) error {
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
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace()
	addFilenameSources(builder, enforceNamespace, filenameOptions, cfg.Streams.In)
	result := builder.
		LabelSelectorParam(cfg.Selector).
		ResourceTypeOrNameArgs(true, cfg.Resources.Args()...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(
		printFlags,
		cfg.Output,
		cfg.Template,
		cfg.AllowMissingTemplateKeys,
		cfg.ShowManagedFields,
		cfg.OutputState,
	)
	if printFlags.OutputFlagSpecified() {
		printer, err := printFlags.ToPrinter()
		if err != nil {
			return err
		}
		return result.Visit(func(info *resource.Info, err error) error {
			if err != nil {
				return err
			}
			historyViewer, err := polymorphichelpers.HistoryViewerFn(factory, info.ResourceMapping())
			if err != nil {
				return err
			}
			history, err := historyViewer.GetHistory(info.Namespace, info.Name)
			if err != nil {
				return err
			}
			if cfg.Revision > 0 {
				revision, ok := history[cfg.Revision]
				if !ok {
					return errRolloutHistoryRevisionNotFound
				}
				return printer.PrintObj(revision, cfg.Streams.Out)
			}
			for _, revision := range slices.Sorted(maps.Keys(history)) {
				if err := printer.PrintObj(history[revision], cfg.Streams.Out); err != nil {
					return err
				}
			}
			return nil
		})
	}

	return result.Visit(func(info *resource.Info, err error) error {
		if err != nil {
			return err
		}
		historyViewer, err := polymorphichelpers.HistoryViewerFn(factory, info.ResourceMapping())
		if err != nil {
			return err
		}
		history, err := historyViewer.ViewHistory(info.Namespace, info.Name, cfg.Revision)
		if err != nil {
			return err
		}
		withRevision := ""
		if cfg.Revision > 0 {
			withRevision = fmt.Sprintf("with revision #%d", cfg.Revision)
		}
		printFlags.NamePrintFlags.Operation = fmt.Sprintf("%s\n%s", withRevision, history)
		printer, err := printFlags.ToPrinter()
		if err != nil {
			return err
		}
		return printer.PrintObj(info.Object, cfg.Streams.Out)
	})
}

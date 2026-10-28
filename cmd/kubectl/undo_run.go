package main

import (
	"context"
	"slices"

	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
	"k8s.io/kubectl/pkg/cmd/util"
)

func runRolloutUndo(_ context.Context, cfg *rolloutUndoConfig) error {
	factory := cfg.Client.factory()
	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options := rollout.NewRolloutUndoOptions(cfg.Streams)
	options.Resources = slices.Clone(cfg.Resources)
	options.ToRevision = cfg.ToRevision
	options.DryRunStrategy = strategy
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filenames),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	options.CmdBaseName = "kubectl"
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace(); err != nil {
		return err
	}
	options.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		options.PrintFlags.NamePrintFlags.Operation = operation
		util.PrintFlagsWithDryRunStrategy(options.PrintFlags, options.DryRunStrategy)
		return options.PrintFlags.ToPrinter()
	}
	options.RESTClientGetter = factory
	options.Builder = factory.NewBuilder
	return options.RunUndo()
}

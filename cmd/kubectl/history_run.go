package main

import (
	"context"
	"slices"

	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
)

func runRolloutHistory(_ context.Context, cfg *rolloutHistoryConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutHistoryOptions(cfg.Streams)
	options.Revision = cfg.Revision
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filenames),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	if err := options.Complete(factory, nil, cfg.Resources); err != nil {
		return err
	}
	return options.Run()
}

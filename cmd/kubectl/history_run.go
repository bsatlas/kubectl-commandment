package main

import (
	"context"

	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
)

func runRolloutHistory(_ context.Context, cfg *rolloutHistoryConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutHistoryOptions(cfg.Streams)
	options.Revision = cfg.Revision
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	configurePrintFlagsWithOutputState(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	if err := options.Complete(factory, nil, cfg.Resources.Args()); err != nil {
		return err
	}
	return options.Run()
}

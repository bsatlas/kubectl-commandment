package main

import (
	"context"
	"slices"

	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
)

func runRolloutStatus(_ context.Context, cfg *rolloutStatusConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutStatusOptions(cfg.Streams)
	options.FilenameOptions = &resource.FilenameOptions{
		Filenames: slices.Clone(cfg.Filenames),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	options.LabelSelector = cfg.Selector
	options.Revision = cfg.Revision
	options.Timeout = cfg.Timeout
	options.Watch = cfg.Watch
	if err := options.Complete(factory, cfg.Resources); err != nil {
		return err
	}
	return options.Run()
}

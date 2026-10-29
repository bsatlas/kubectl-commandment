package main

import (
	"context"

	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
)

func runRolloutStatus(_ context.Context, cfg *rolloutStatusConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutStatusOptions(cfg.Streams)
	options.FilenameOptions = &resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	options.LabelSelector = cfg.Selector
	options.Revision = cfg.Revision
	options.Timeout = cfg.Timeout
	options.Watch = cfg.Watch
	if err := options.Complete(factory, cfg.Resources.Args()); err != nil {
		return err
	}
	return options.Run()
}

package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/describe"
)

func runDescribe(_ context.Context, cfg *describeConfig) error {
	flags := describe.NewDescribeFlags(cfg.Client.factory(), cfg.Streams)
	flags.FilenameOptions.Filenames = append([]string(nil), cfg.Filename...)
	flags.FilenameOptions.Kustomize = cfg.Kustomize
	flags.FilenameOptions.Recursive = cfg.Recursive
	flags.Selector = cfg.Selector
	flags.AllNamespaces = cfg.AllNamespaces
	flags.DescriberSettings.ShowEvents = cfg.ShowEvents
	flags.DescriberSettings.ChunkSize = cfg.ChunkSize
	flags.ShowEventsExplicitlySet = explicitFlagValue(cfg.ShowEventsState)
	options, err := flags.ToOptions("kubectl", cfg.Arguments)
	if err != nil {
		return err
	}
	return options.Run()
}

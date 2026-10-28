package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/explain"
)

func runExplain(_ context.Context, cfg *explainConfig) error {
	flags := explain.NewExplainFlags(cfg.Streams)
	flags.Recursive = cfg.Recursive
	flags.APIVersion = cfg.APIVersion
	flags.OutputFormat = cfg.Output
	flags.MaxDepth = cfg.MaxDepth
	options, err := flags.ToOptions(cfg.Client.factory(), "kubectl", []string{cfg.Resource})
	if err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

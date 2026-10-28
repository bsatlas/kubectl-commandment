package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/top"
)

func runTopNode(_ context.Context, cfg *topNodeConfig) error {
	flags := top.NewTopNodeFlags(cfg.Streams)
	flags.Selector = cfg.Selector
	flags.SortBy = cfg.SortBy
	flags.NoHeaders = cfg.NoHeaders
	flags.UseProtocolBuffers = cfg.UseProtocolBuffers
	flags.ShowCapacity = cfg.ShowCapacity
	flags.ShowSwap = cfg.ShowSwap
	options, err := flags.ToOptions(cfg.Client.factory(), cfg.Arguments)
	if err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunTopNode()
}

func runTopPod(_ context.Context, cfg *topPodConfig) error {
	flags := top.NewTopPodFlags(cfg.Streams)
	flags.LabelSelector = cfg.Selector
	flags.FieldSelector = cfg.FieldSelector
	flags.SortBy = cfg.SortBy
	flags.PrintContainers = cfg.Containers
	flags.AllNamespaces = cfg.AllNamespaces
	flags.NoHeaders = cfg.NoHeaders
	flags.UseProtocolBuffers = cfg.UseProtocolBuffers
	flags.Sum = cfg.Sum
	flags.ShowSwap = cfg.ShowSwap
	options, err := flags.ToOptions(cfg.Client.factory(), cfg.Arguments)
	if err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunTopPod()
}

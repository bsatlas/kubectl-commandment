package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/events"
)

func runEvents(_ context.Context, cfg *eventsConfig) error {
	flags := events.NewEventsFlags(restClientGetter(cfg.Client), cfg.Streams)
	flags.AllNamespaces = cfg.AllNamespaces
	flags.Watch = cfg.Watch
	flags.NoHeaders = cfg.NoHeaders
	flags.ForObject = cfg.For
	flags.FilterTypes = append([]string(nil), cfg.Types...)
	flags.ChunkSize = cfg.ChunkSize
	configurePrintFlags(flags.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options, err := flags.ToOptions()
	if err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}
	return options.Run()
}

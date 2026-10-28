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
	if cfg.For.Resource != "" {
		flags.ForObject = cfg.For.Resource + "/" + cfg.For.Name
	}
	flags.FilterTypes = append([]string(nil), cfg.Types...)
	flags.ChunkSize = cfg.ChunkSize
	configurePrintFlags(flags.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	flags.PrintFlags.OutputFlagSpecified = func() bool { return explicitFlagValue(cfg.OutputState) }

	options, err := flags.ToOptions()
	if err != nil {
		return err
	}
	return options.Run()
}

package main

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/wait"
)

func runWait(ctx context.Context, cfg *waitConfig) error {
	flags := wait.NewWaitFlags(restClientGetter(cfg.Client), cfg.Streams)
	flags.Timeout = cfg.Timeout
	flags.ForCondition = append([]string(nil), cfg.For...)
	flags.ResourceBuilderFlags.
		WithFile(cfg.Recursive, nil, cfg.Filename...).
		WithLabelSelector(cfg.Selector).
		WithFieldSelector(cfg.FieldSelector).
		WithAllNamespaces(cfg.AllNamespaces).
		WithAll(cfg.All).
		WithLocal(cfg.Local)
	configurePrintFlags(flags.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options, err := flags.ToOptions(cfg.Arguments)
	if err != nil {
		return err
	}
	return options.RunWaitContext(ctx)
}

package main

import (
	"context"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	"k8s.io/kubectl/pkg/cmd/auth"
)

func runAuthWhoAmI(ctx context.Context, cfg *authWhoAmIConfig) error {
	streams := genericiooptions.IOStreams{
		Out: cfg.Stdout,
	}
	flags := auth.NewWhoAmIFlags(restClientGetterForInvocation(ctx, cfg.Client), streams)
	configurePrintFlagsWithOutputState(flags.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	options, err := flags.ToOptions(ctx, nil)
	if err != nil {
		return err
	}
	return options.Run()
}

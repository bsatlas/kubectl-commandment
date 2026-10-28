package main

import (
	"context"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

func completeResourceType(_ context.Context, cfg *getConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	return completeResourceTypes(restClientGetter(cfg.Client), resourceCompletionBare)
}

func completeResourceName(resourceType cli.Destination[getConfig, string]) cli.ValueCompleter[getConfig] {
	return func(ctx context.Context, cfg *getConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		state, ok := cli.ValueStateFromContext(ctx, resourceType)
		if !ok || state.State() != cli.ResolutionStateResolved || cfg.ResourceType == "" {
			return cli.CompletionContribution{}, nil
		}
		return completeResourceNames(ctx, restClientGetter(cfg.Client), cfg.ResourceType, resourceCompletionBare)
	}
}

func completeQualifiedResource(ctx context.Context, cfg *getConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	resourceType, _, found := strings.Cut(request.Text, "/")
	if !found {
		return completeResourceTypes(restClientGetter(cfg.Client), resourceCompletionQualified)
	}
	return completeResourceNames(ctx, restClientGetter(cfg.Client), resourceType, resourceCompletionQualified)
}

package main

import (
	"context"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

func literalCompleter[C any](kind cli.CompletionCandidateKind, values ...string) cli.ValueCompleter[C] {
	return func(_ context.Context, _ *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		candidates := make([]cli.CompletionCandidate, 0, len(values))
		for _, value := range values {
			if !strings.HasPrefix(value, request.Text) {
				continue
			}
			candidates = append(candidates, cli.CompletionCandidate{
				Value: value,
				Kind:  kind,
			})
		}
		completion := cli.CompletionContribution{
			Candidates: candidates,
		}
		return completion, nil
	}
}

func completionFromStrings(values []string, kind cli.CompletionCandidateKind, spacing cli.CompletionSpacing) cli.CompletionContribution {
	candidates := make([]cli.CompletionCandidate, 0, len(values))
	for _, value := range values {
		candidates = append(candidates, cli.CompletionCandidate{
			Value:   value,
			Kind:    kind,
			Spacing: spacing,
		})
	}
	completion := cli.CompletionContribution{
		Candidates: candidates,
	}
	return completion
}

package kubectl

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
)

type resourceArguments struct {
	Type      string
	Names     []string
	Qualified []string
}

func (arguments resourceArguments) Args() []string {
	if len(arguments.Qualified) != 0 {
		return slices.Clone(arguments.Qualified)
	}
	if arguments.Type == "" {
		return nil
	}
	args := make([]string, 1, 1+len(arguments.Names))
	args[0] = arguments.Type
	return append(args, arguments.Names...)
}

type resourceArgumentDestinations[C any] struct {
	Type      cli.Destination[C, string]
	Qualified cli.Destination[C, []string]
}

func requireResourceOrFilenameSource[C any](
	cb *cli.CommandBuilder[C],
	resources resourceArgumentDestinations[C],
	files filenameOptionDestinations[C],
) {
	cb.Rule.Check("kubectl-resource-input-source", func(_ context.Context, _ *C, dependencies cli.Dependencies) (bool, error) {
		resourceType, _ := dependencies.Get(resources.Type)
		qualified, _ := dependencies.Get(resources.Qualified)
		filenameGroups, _ := dependencies.Get(files.Filenames)
		kustomize, _ := dependencies.Get(files.Kustomize)
		return resourceType != "" || len(qualified) != 0 || len(slices.Concat(filenameGroups...)) != 0 || kustomize != "", nil
	}).
		Subject(resources.Type).
		Subject(resources.Qualified).
		Subject(files.Filenames).
		Subject(files.Kustomize).
		DependsOn(resources.Type.OptionalDependency()).
		DependsOn(resources.Qualified.OptionalDependency()).
		DependsOn(files.Filenames.OptionalDependency()).
		DependsOn(files.Kustomize.OptionalDependency())
}

func declareResourceArguments[C any](
	cb *cli.CommandBuilder[C],
	storage *resourceArguments,
	client cli.Destination[C, clientConfig],
	access func(*C) (clientConfig, resourceArguments),
	allowedTypes []string,
) resourceArgumentDestinations[C] {
	resourceType := cb.Values.Value(&storage.Type, resourceTypeLanguage)
	names := cb.Values.Sequence(&storage.Names, resourceNameLanguage)
	qualified := cb.Values.Sequence(&storage.Qualified, qualifiedResourceLanguage)

	cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[C]) {})
	cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[C]) {
		form.ValueFor(resourceType, "TYPE").
			CompleteBy(func(_ context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				clientConfig, _ := access(cfg)
				return completeAllowedResourceTypes(restClientGetter(clientConfig), allowedTypes, request.Text, resourceCompletionBare)
			}).
			From(client)
		form.Sequence(names, "NAME", cli.Count.ZeroOrMore()).
			CompleteBy(func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				clientConfig, arguments := access(cfg)
				if arguments.Type == "" {
					return cli.CompletionContribution{}, nil
				}
				completion, err := completeResourceNames(ctx, restClientGetter(clientConfig), arguments.Type, resourceCompletionBare)
				if err != nil {
					return cli.CompletionContribution{}, err
				}
				completion.Candidates = filterCompletionCandidates(completion.Candidates, request.Text, arguments.Names)
				return completion, nil
			}).
			From(client, resourceType.OptionalDependency())
	})
	cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[C]) {
		form.Sequence(qualified, "TYPE/NAME", cli.Count.OneOrMore()).
			CompleteBy(func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
				clientConfig, arguments := access(cfg)
				resourceType, _, found := strings.Cut(request.Text, "/")
				if !found {
					return completeAllowedResourceTypes(restClientGetter(clientConfig), allowedTypes, request.Text, resourceCompletionQualified)
				}
				completion, err := completeResourceNames(ctx, restClientGetter(clientConfig), resourceType, resourceCompletionQualified)
				if err != nil {
					return cli.CompletionContribution{}, err
				}
				completion.Candidates = filterCompletionCandidates(completion.Candidates, request.Text, arguments.Qualified)
				return completion, nil
			}).
			From(client)
	})

	destinations := resourceArgumentDestinations[C]{
		Type:      resourceType,
		Qualified: qualified,
	}
	return destinations
}

var resourceTypeLanguage = cli.ValueCodec[string]{
	Name: "kubectl-resource-type",
	Parse: func(value string) (string, error) {
		if value == "" || strings.Contains(value, "/") {
			return "", cli.Rejection.Message(
				"resource-type",
				fmt.Sprintf("expected resource type, got %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

var resourceNameLanguage = cli.ValueCodec[string]{
	Name: "kubectl-resource-name",
	Parse: func(value string) (string, error) {
		if strings.Contains(value, "/") {
			return "", cli.Rejection.Message(
				"resource-name",
				fmt.Sprintf("expected resource name, got %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

var qualifiedResourceLanguage = cli.ValueCodec[string]{
	Name: "kubectl-qualified-resource",
	Parse: func(value string) (string, error) {
		resourceType, name, found := strings.Cut(value, "/")
		if !found || resourceType == "" || name == "" || strings.Contains(resourceType, ",") || strings.Contains(name, "/") {
			return "", cli.Rejection.Message(
				"qualified-resource",
				fmt.Sprintf("expected TYPE/NAME, got %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

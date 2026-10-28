package main

import (
	"context"
	"sort"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

type resourceCompletionInputs[C any] func(*C) (clientConfig, []string)

type resourceCompletionForm uint8

const (
	resourceCompletionBare resourceCompletionForm = iota
	resourceCompletionQualified
)

type resourceArgumentRepetition uint8

const (
	resourceArgumentSingle resourceArgumentRepetition = iota
	resourceArgumentRepeated
)

type resourceDiscoveryMode uint8

const (
	resourceDiscoveryAll resourceDiscoveryMode = iota
	resourceDiscoveryReadable
)

func resourceArgumentCompleter[C any](inputs resourceCompletionInputs[C], allowedTypes []string, repetition resourceArgumentRepetition) cli.ValueCompleter[C] {
	return func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		client, arguments := inputs(cfg)
		return completeResourceArguments(ctx, client, arguments, request, allowedTypes, repetition)
	}
}

func fixedResourceArgumentCompleter[C any](inputs resourceCompletionInputs[C], resourceType string) cli.ValueCompleter[C] {
	return func(ctx context.Context, cfg *C, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		client, arguments := inputs(cfg)
		if len(arguments) != 0 {
			return cli.CompletionContribution{}, nil
		}
		completion, err := completeResourceNames(ctx, restClientGetter(client), resourceType, resourceCompletionBare)
		if err != nil {
			return cli.CompletionContribution{}, err
		}
		completion.Candidates = filterCompletionCandidates(completion.Candidates, request.Text, nil)
		return completion, nil
	}
}

func completeResourceArguments(ctx context.Context, client clientConfig, arguments []string, request cli.ValueCompletionRequest, allowedTypes []string, repetition resourceArgumentRepetition) (cli.CompletionContribution, error) {
	if len(arguments) > 0 && !strings.Contains(arguments[0], "/") {
		if repetition == resourceArgumentSingle && len(arguments) != 1 {
			return cli.CompletionContribution{}, nil
		}
		completion, err := completeResourceNames(ctx, restClientGetter(client), arguments[0], resourceCompletionBare)
		if err != nil {
			return cli.CompletionContribution{}, err
		}
		completion.Candidates = filterCompletionCandidates(completion.Candidates, request.Text, arguments[1:])
		return completion, nil
	}

	slash := strings.Index(request.Text, "/")
	if slash < 0 {
		if len(arguments) == 0 {
			return completeAllowedResourceTypes(restClientGetter(client), allowedTypes, request.Text, resourceCompletionBare)
		}
		if repetition == resourceArgumentSingle {
			return cli.CompletionContribution{}, nil
		}
		return completeAllowedResourceTypes(restClientGetter(client), allowedTypes, request.Text, resourceCompletionQualified)
	}
	if repetition == resourceArgumentSingle && len(arguments) != 0 {
		return cli.CompletionContribution{}, nil
	}

	resourceType := request.Text[:slash]
	prefix := request.Text[slash+1:]
	completion, err := completeResourceNames(ctx, restClientGetter(client), resourceType, resourceCompletionQualified)
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	completion.Candidates = filterCompletionCandidates(completion.Candidates, resourceType+"/"+prefix, arguments)
	return completion, nil
}

func completeAllowedResourceTypes(getter genericclioptions.RESTClientGetter, allowedTypes []string, prefix string, form resourceCompletionForm) (cli.CompletionContribution, error) {
	if len(allowedTypes) == 0 {
		completion, err := completeResourceTypes(getter, form, prefix)
		if err != nil {
			return cli.CompletionContribution{}, err
		}
		if form == resourceCompletionQualified {
			completion.Spacing = cli.CompletionSpacingNoSpace
		}
		return completion, nil
	}

	candidates := make([]cli.CompletionCandidate, 0, len(allowedTypes))
	for _, resourceType := range allowedTypes {
		if !strings.HasPrefix(resourceType, prefix) {
			continue
		}
		value := resourceType
		spacing := cli.CompletionSpacingDefault
		if form == resourceCompletionQualified {
			value += "/"
			spacing = cli.CompletionSpacingNoSpace
		}
		candidates = append(candidates, cli.CompletionCandidate{
			Value:   value,
			Kind:    cli.CompletionCandidateResource,
			Spacing: spacing,
		})
	}
	completion := cli.CompletionContribution{
		Candidates: candidates,
	}
	return completion, nil
}

func completeResourceTypes(getter genericclioptions.RESTClientGetter, form resourceCompletionForm, prefix string) (cli.CompletionContribution, error) {
	names, err := discoveredResourceTypeNames(getter, resourceDiscoveryReadable)
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	completedPrefix := ""
	suffix := prefix
	if comma := strings.LastIndex(prefix, ","); comma >= 0 {
		completedPrefix = prefix[:comma+1]
		suffix = prefix[comma+1:]
	}
	candidates := make([]cli.CompletionCandidate, 0, len(names))
	for _, name := range names {
		if !strings.HasPrefix(name, suffix) {
			continue
		}
		value := completedPrefix + name
		spacing := cli.CompletionSpacingDefault
		if form == resourceCompletionQualified {
			value += "/"
			spacing = cli.CompletionSpacingNoSpace
		}
		candidates = append(candidates, cli.CompletionCandidate{
			Value:   value,
			Kind:    cli.CompletionCandidateResource,
			Spacing: spacing,
		})
	}
	completion := cli.CompletionContribution{
		Candidates: candidates,
	}
	return completion, nil
}

func completeResourceNames(ctx context.Context, getter genericclioptions.RESTClientGetter, resourceType string, form resourceCompletionForm) (cli.CompletionContribution, error) {
	namespace, _, err := getter.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	return completeResourceNamesInNamespace(ctx, getter, resourceType, namespace, form)
}

func completeResourceNamesInNamespace(ctx context.Context, getter genericclioptions.RESTClientGetter, resourceType, namespace string, form resourceCompletionForm) (cli.CompletionContribution, error) {
	result := resource.NewBuilder(getter).
		Unstructured().
		NamespaceParam(namespace).
		DefaultNamespace().
		ResourceTypeOrNameArgs(true, resourceType).
		ContinueOnError().
		Latest().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return cli.CompletionContribution{}, err
	}
	infos, err := result.Infos()
	if err != nil {
		return cli.CompletionContribution{}, err
	}

	candidates := make([]cli.CompletionCandidate, 0, len(infos))
	for _, info := range infos {
		if err := ctx.Err(); err != nil {
			return cli.CompletionContribution{}, err
		}
		value := info.Name
		if form == resourceCompletionQualified {
			value = resourceType + "/" + value
		}
		candidates = append(candidates, cli.CompletionCandidate{
			Value: value,
			Kind:  cli.CompletionCandidateResource,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Value < candidates[j].Value
	})
	completion := cli.CompletionContribution{
		Candidates: candidates,
	}
	return completion, nil
}

func discoveredResourceTypeNames(getter genericclioptions.RESTClientGetter, mode resourceDiscoveryMode) ([]string, error) {
	discoveryClient, err := getter.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	lists, discoveryErr := discoveryClient.ServerPreferredResources()
	if discoveryErr != nil && len(lists) == 0 {
		return nil, discoveryErr
	}

	values := make(map[string]struct{})
	for _, list := range lists {
		group := ""
		if groupVersion, parseErr := schema.ParseGroupVersion(list.GroupVersion); parseErr == nil {
			group = groupVersion.Group
		}
		for _, apiResource := range list.APIResources {
			if strings.Contains(apiResource.Name, "/") || mode == resourceDiscoveryReadable && !supportsGet(apiResource.Verbs) {
				continue
			}
			name := apiResource.Name
			if group != "" {
				name += "." + group
			}
			values[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func filterCompletionCandidates(candidates []cli.CompletionCandidate, prefix string, excluded []string) []cli.CompletionCandidate {
	if len(candidates) == 0 {
		return nil
	}
	exclude := make(map[string]struct{}, len(excluded))
	for _, value := range excluded {
		exclude[value] = struct{}{}
	}
	filtered := candidates[:0]
	for _, candidate := range candidates {
		if !strings.HasPrefix(candidate.Value, prefix) {
			continue
		}
		if _, exists := exclude[candidate.Value]; exists {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func supportsGet(verbs []string) bool {
	for _, verb := range verbs {
		if verb == "get" || verb == "list" {
			return true
		}
	}
	return false
}

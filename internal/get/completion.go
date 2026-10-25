package get

import (
	"context"
	"sort"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
)

func completeResourceType(services Services) cli.ValueCompleter[commandConfig] {
	return func(_ context.Context, cfg *commandConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		return completeResourceTypes(services.restClientGetter(cfg.Client), false)
	}
}

func completeResourceName(services Services, resourceType cli.Destination[commandConfig, string]) cli.ValueCompleter[commandConfig] {
	return func(ctx context.Context, cfg *commandConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		state, ok := cli.ValueStateFromContext(ctx, resourceType)
		if !ok || state.State() != cli.ResolutionStateResolved || cfg.ResourceType == "" {
			return cli.CompletionContribution{}, nil
		}
		return completeResourceNames(ctx, services.restClientGetter(cfg.Client), cfg.ResourceType, false)
	}
}

func completeQualifiedResource(services Services) cli.ValueCompleter[commandConfig] {
	return func(ctx context.Context, cfg *commandConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
		resourceType, _, found := strings.Cut(request.Text, "/")
		if !found {
			return completeResourceTypes(services.restClientGetter(cfg.Client), true)
		}
		return completeResourceNames(ctx, services.restClientGetter(cfg.Client), resourceType, true)
	}
}

func completeResourceTypes(getter genericclioptions.RESTClientGetter, qualify bool) (cli.CompletionContribution, error) {
	discoveryClient, err := getter.ToDiscoveryClient()
	if err != nil {
		return cli.CompletionContribution{}, err
	}
	lists, discoveryErr := discoveryClient.ServerPreferredResources()
	if discoveryErr != nil && len(lists) == 0 {
		return cli.CompletionContribution{}, discoveryErr
	}

	values := make(map[string]struct{})
	for _, list := range lists {
		group := ""
		if groupVersion, parseErr := schema.ParseGroupVersion(list.GroupVersion); parseErr == nil {
			group = groupVersion.Group
		}
		for _, apiResource := range list.APIResources {
			if strings.Contains(apiResource.Name, "/") || !supportsGet(apiResource.Verbs) {
				continue
			}
			name := apiResource.Name
			if group != "" {
				name += "." + group
			}
			values[name] = struct{}{}
			for _, shortName := range apiResource.ShortNames {
				values[shortName] = struct{}{}
			}
		}
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	candidates := make([]cli.CompletionCandidate, 0, len(names))
	for _, name := range names {
		if qualify {
			name += "/"
		}
		candidates = append(candidates, cli.CompletionCandidate{
			Value: name,
			Kind:  cli.CompletionCandidateResource,
		})
	}
	return cli.CompletionContribution{Candidates: candidates}, nil
}

func completeResourceNames(ctx context.Context, getter genericclioptions.RESTClientGetter, resourceType string, qualified bool) (cli.CompletionContribution, error) {
	namespace, _, err := getter.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return cli.CompletionContribution{}, err
	}
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
		if qualified {
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
	return cli.CompletionContribution{Candidates: candidates}, nil
}

func supportsGet(verbs []string) bool {
	for _, verb := range verbs {
		if verb == "get" || verb == "list" {
			return true
		}
	}
	return false
}

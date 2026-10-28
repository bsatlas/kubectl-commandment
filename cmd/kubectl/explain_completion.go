package main

import (
	"context"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/openapi3"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/kubectl/pkg/explain"
)

const explainSchemaRefPrefix = "#/components/schemas/"

func completeExplainResource(_ context.Context, cfg *explainConfig, request cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	if strings.Contains(request.Text, "..") {
		return cli.CompletionContribution{}, nil
	}

	getter := restClientGetter(cfg.Client)
	if !strings.Contains(request.Text, ".") {
		names, err := discoveredResourceTypeNames(getter, resourceDiscoveryAll)
		if err != nil {
			return cli.CompletionContribution{}, err
		}
		candidates := make([]cli.CompletionCandidate, 0, len(names))
		for _, name := range names {
			if !strings.HasPrefix(name, request.Text) {
				continue
			}
			candidates = append(candidates, cli.CompletionCandidate{
				Value:   name + ".",
				Kind:    cli.CompletionCandidateResource,
				Spacing: cli.CompletionSpacingNoSpace,
			})
		}
		completion := cli.CompletionContribution{
			Candidates: candidates,
			Spacing:    cli.CompletionSpacingNoSpace,
		}
		return completion, nil
	}

	mapper, err := getter.ToRESTMapper()
	if err != nil {
		return cli.CompletionContribution{}, nil
	}

	lastDot := strings.LastIndex(request.Text, ".")
	path := request.Text[:lastDot]
	prefix := request.Text[lastDot+1:]

	var (
		gvr        schema.GroupVersionResource
		fieldsPath []string
	)
	if cfg.APIVersion == "" {
		gvr, fieldsPath, err = explain.SplitAndParseResourceRequestWithMatchingPrefix(path, mapper)
	} else {
		gvr, fieldsPath, err = explain.SplitAndParseResourceRequest(path, mapper)
	}
	if err != nil {
		return cli.CompletionContribution{}, nil
	}

	if cfg.APIVersion != "" {
		groupVersion, parseErr := schema.ParseGroupVersion(cfg.APIVersion)
		if parseErr != nil {
			return cli.CompletionContribution{}, nil
		}
		gvr.Group = groupVersion.Group
		gvr.Version = groupVersion.Version
	}

	candidates := make([]cli.CompletionCandidate, 0)
	hasExpandable := false
	expandable, leaves := explainFieldNamesForGVR(getter, mapper, gvr, fieldsPath)
	base := request.Text[:lastDot+1]
	for _, name := range expandable {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		candidates = append(candidates, cli.CompletionCandidate{
			Value:   base + name + ".",
			Kind:    cli.CompletionCandidateValue,
			Spacing: cli.CompletionSpacingNoSpace,
		})
		hasExpandable = true
	}
	for _, name := range leaves {
		if strings.HasPrefix(name, prefix) {
			candidates = append(candidates, cli.CompletionCandidate{
				Value: base + name,
				Kind:  cli.CompletionCandidateValue,
			})
		}
	}

	if cfg.APIVersion == "" {
		resource, _, _ := strings.Cut(request.Text, ".")
		groupVersionResources, _ := mapper.ResourcesFor(schema.GroupVersionResource{
			Resource: resource,
		})
		seen := make(map[string]struct{})
		for _, candidate := range groupVersionResources {
			groupResource := candidate.GroupResource().String()
			if _, exists := seen[groupResource]; exists || !strings.HasPrefix(groupResource, request.Text) {
				continue
			}
			seen[groupResource] = struct{}{}
			selected, fields, parseErr := explain.SplitAndParseResourceRequestWithMatchingPrefix(groupResource, mapper)
			if parseErr != nil || len(fields) != 0 || selected.GroupResource() != candidate.GroupResource() {
				continue
			}
			candidates = append(candidates, cli.CompletionCandidate{
				Value:   groupResource + ".",
				Kind:    cli.CompletionCandidateResource,
				Spacing: cli.CompletionSpacingNoSpace,
			})
			hasExpandable = true
		}
	}

	completion := cli.CompletionContribution{
		Candidates: candidates,
	}
	if hasExpandable {
		completion.Spacing = cli.CompletionSpacingNoSpace
	}
	return completion, nil
}

func explainFieldNamesForGVR(
	getter genericclioptions.RESTClientGetter,
	mapper meta.RESTMapper,
	gvr schema.GroupVersionResource,
	fieldsPath []string,
) (expandable, leaves []string) {
	gvk, err := mapper.KindFor(gvr)
	if err != nil || gvk.Empty() {
		preferred, preferredErr := mapper.KindFor(gvr.GroupResource().WithVersion(""))
		if preferredErr != nil || preferred.Empty() {
			return nil, nil
		}
		gvk = gvr.GroupVersion().WithKind(preferred.Kind)
	}

	discoveryClient, err := getter.ToDiscoveryClient()
	if err != nil {
		return nil, nil
	}
	groupVersionSpec, err := openapi3.NewRoot(discoveryClient.OpenAPIV3()).GVSpec(gvk.GroupVersion())
	if err != nil || groupVersionSpec.Components == nil {
		return nil, nil
	}
	schemas := groupVersionSpec.Components.Schemas

	current := explainSchemaForGVK(schemas, gvk)
	for _, field := range fieldsPath {
		object := resolveExplainObject(current, schemas, map[string]bool{})
		if object == nil {
			return nil, nil
		}
		next, ok := object.Properties[field]
		if !ok {
			return nil, nil
		}
		current = &next
	}

	object := resolveExplainObject(current, schemas, map[string]bool{})
	if object == nil {
		return nil, nil
	}
	for name, field := range object.Properties {
		if resolveExplainObject(&field, schemas, map[string]bool{}) != nil {
			expandable = append(expandable, name)
		} else {
			leaves = append(leaves, name)
		}
	}
	slices.Sort(expandable)
	slices.Sort(leaves)
	return expandable, leaves
}

func explainSchemaForGVK(schemas map[string]*spec.Schema, gvk schema.GroupVersionKind) *spec.Schema {
	for _, candidate := range schemas {
		values, ok := candidate.Extensions["x-kubernetes-group-version-kind"].([]interface{})
		if !ok {
			continue
		}
		for _, value := range values {
			identity, ok := value.(map[string]interface{})
			if !ok {
				continue
			}
			if identity["group"] == gvk.Group && identity["version"] == gvk.Version && identity["kind"] == gvk.Kind {
				return candidate
			}
		}
	}
	return nil
}

func resolveExplainObject(value *spec.Schema, schemas map[string]*spec.Schema, visited map[string]bool) *spec.Schema {
	if value == nil {
		return nil
	}
	if ref, ok := explainReferencedSchemaName(value); ok {
		if visited[ref] {
			return nil
		}
		visited[ref] = true
		return resolveExplainObject(schemas[ref], schemas, visited)
	}
	if value.Items != nil && value.Items.Schema != nil {
		return resolveExplainObject(value.Items.Schema, schemas, visited)
	}
	if len(value.Properties) > 0 {
		return value
	}
	return nil
}

func explainReferencedSchemaName(value *spec.Schema) (string, bool) {
	ref := value.Ref.String()
	if ref == "" && len(value.AllOf) == 1 {
		ref = value.AllOf[0].Ref.String()
	}
	return strings.CutPrefix(ref, explainSchemaRefPrefix)
}

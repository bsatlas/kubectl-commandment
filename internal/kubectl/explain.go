package kubectl

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/openapi3"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/kubectl/pkg/explain"
)

type explainConfig struct {
	Client     clientConfig
	Streams    genericiooptions.IOStreams
	Resource   string
	APIVersion schema.GroupVersion
	MaxDepth   int
	Output     string
	Recursive  bool
}

func explainCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[explainConfig] {
	cmd := cli.Command[explainConfig]{
		Name: "explain",
		Description: cli.Description{
			Short: "Get documentation for a resource",
			Long:  "Describe fields and structure of various resources.\n\n This command describes the fields associated with each supported API resource. Fields are identified via a simple JSONPath identifier:\n\n        TYPE.FIELDNAME[.FIELDNAME]\n        \n Information about each field is retrieved from the server in OpenAPI format.\n\nUse \"kubectl api-resources\" for a complete list of supported resources.",
		},
		Examples: []cli.Example{
			{
				Description: "Get the documentation of the resource and its fields",
				CommandLine: "kubectl explain pods",
			},
			{
				Description: "Get all the fields in the resource",
				CommandLine: "kubectl explain pods --recursive",
			},
			{
				Description: "Get fields in the resource up to a specific recursive depth",
				CommandLine: "kubectl explain pods --recursive --max-depth=2",
			},
			{
				Description: "Get the explanation for deployment in supported api versions",
				CommandLine: "kubectl explain deployments --api-version=apps/v1",
			},
			{
				Description: "Get the documentation of a specific field of a resource",
				CommandLine: "kubectl explain pods.spec.containers",
			},
			{
				Description: "Get the documentation of resources in different format",
				CommandLine: "kubectl explain deployment --output=plaintext-openapiv2",
			},
		},
		Declare: func(cb *cli.CommandBuilder[explainConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			apiVersion := cb.Values.Value(&cb.Config.APIVersion, apiVersionLanguage)
			apiVersionFlag := cb.Flags.ValueFor(apiVersion)
			apiVersionFlag.Description("Get different explanations for particular API version (API group/version)").Long("api-version")

			resource := cb.Values.String(&cb.Config.Resource)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[explainConfig]) {
				form.ValueFor(resource, "TYPE").
					CompleteBy(completeExplainResource).
					From(client, apiVersion.OptionalDependency())
			})

			maxDepthFlag := cb.Flags.Int(&cb.Config.MaxDepth)
			maxDepthFlag.Description("Maximum recursion depth when printing nested fields with --recursive. 0 means no limit. Requires --recursive when greater than 0.").Long("max-depth")
			cb.Rule.AtLeast(maxDepthFlag.Destination(), 0)

			output := cb.Values.String(&cb.Config.Output)
			cb.Values.Default(output, "plaintext")
			outputFlag := cb.Flags.ValueFor(output)
			outputFlag.Description("Format in which to render the schema (plaintext, plaintext-openapiv2)").Long("output")
			outputFlag.Short('o')
			cb.Rule.In(output, "plaintext", "plaintext-openapiv2")

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag.Description("Print the fields of fields. Use --max-depth to cap the recursion depth.").Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			cb.Rule.Check("kubectl-explain-recursive-depth", func(_ context.Context, cfg *explainConfig, _ cli.Dependencies) (bool, error) {
				return cfg.MaxDepth == 0 || cfg.Recursive, nil
			}).
				DependsOn(maxDepthFlag.Destination().OptionalDependency()).
				DependsOn(recursiveFlag.Destination().OptionalDependency())
		},
		Run: runExplain,
	}
	return cmd
}

var apiVersionLanguage = cli.ValueCodec[schema.GroupVersion]{
	Name: "kubernetes-api-version",
	Parse: func(value string) (schema.GroupVersion, error) {
		if value == "" {
			return schema.GroupVersion{}, nil
		}
		groupVersion, err := schema.ParseGroupVersion(value)
		if err == nil && groupVersion.Version != "" {
			return groupVersion, nil
		}
		return schema.GroupVersion{}, cli.Rejection.Message(
			"api-version",
			fmt.Sprintf("expected API group/version, got %q", value),
			cli.Rejection.At(0, 0, len(value)),
		)
	},
	Format: func(value schema.GroupVersion) string {
		if value.Version == "" {
			return ""
		}
		return value.String()
	},
	Snapshot: func(value schema.GroupVersion) schema.GroupVersion { return value },
}

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
	if cfg.APIVersion.Version == "" {
		gvr, fieldsPath, err = explain.SplitAndParseResourceRequestWithMatchingPrefix(path, mapper)
	} else {
		gvr, fieldsPath, err = explain.SplitAndParseResourceRequest(path, mapper)
	}
	if err != nil {
		return cli.CompletionContribution{}, nil
	}

	if cfg.APIVersion.Version != "" {
		gvr.Group = cfg.APIVersion.Group
		gvr.Version = cfg.APIVersion.Version
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

	if cfg.APIVersion.Version == "" {
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

func explainFieldNamesForGVR(getter genericclioptions.RESTClientGetter, mapper meta.RESTMapper, gvr schema.GroupVersionResource, fieldsPath []string) (expandable, leaves []string) {
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

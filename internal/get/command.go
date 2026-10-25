package get

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/kubectl-commandment/internal/root"
)

var outputFormats = []string{
	"json",
	"yaml",
	"kyaml",
	"name",
	"go-template",
	"go-template-file",
	"template",
	"templatefile",
	"jsonpath",
	"jsonpath-as-json",
	"jsonpath-file",
	"custom-columns",
	"custom-columns-file",
	"wide",
}

var parameterizedOutputFormats = map[string]struct{}{
	"go-template":         {},
	"go-template-file":    {},
	"template":            {},
	"templatefile":        {},
	"jsonpath":            {},
	"jsonpath-as-json":    {},
	"jsonpath-file":       {},
	"custom-columns":      {},
	"custom-columns-file": {},
}

func resourceTypeLanguage() cli.ValueCodec[string] {
	return cli.ValueCodec[string]{
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
}

func qualifiedResourceLanguage() cli.ValueCodec[string] {
	return cli.ValueCodec[string]{
		Name: "kubectl-qualified-resource",
		Parse: func(value string) (string, error) {
			resourceType, name, found := strings.Cut(value, "/")
			if !found || resourceType == "" || name == "" {
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
}

func outputValueLanguage() cli.ValueCodec[string] {
	return cli.ValueCodec[string]{
		Name: "kubectl-output-format",
		Parse: func(value string) (string, error) {
			if value == "" {
				return value, nil
			}
			for _, format := range outputFormats {
				if value == format {
					return value, nil
				}
			}
			format, payload, found := strings.Cut(value, "=")
			if _, ok := parameterizedOutputFormats[format]; ok && found && payload != "" {
				return value, nil
			}
			return "", cli.Rejection.Message(
				"output-format",
				fmt.Sprintf("unsupported output format %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		},
		Format:   func(value string) string { return value },
		Snapshot: func(value string) string { return value },
	}
}

func int64Language() cli.ValueCodec[int64] {
	return cli.ValueCodec[int64]{
		Name: "int64",
		Parse: func(value string) (int64, error) {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, cli.Rejection.Message(
					"int64",
					fmt.Sprintf("expected signed integer, got %q", value),
					cli.Rejection.At(0, 0, len(value)),
				)
			}
			return parsed, nil
		},
		Format:   func(value int64) string { return strconv.FormatInt(value, 10) },
		Snapshot: func(value int64) int64 { return value },
	}
}

func completeOutputFormat(_ context.Context, _ *commandConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	candidates := make([]cli.CompletionCandidate, 0, len(outputFormats))
	for _, format := range outputFormats {
		candidates = append(candidates, cli.CompletionCandidate{
			Value: format,
			Kind:  cli.CompletionCandidateValue,
		})
	}
	return cli.CompletionContribution{Candidates: candidates}, nil
}

// Command returns kubectl's get command with root-owned client inputs bound explicitly.
func Command(
	kubeconfig cli.Destination[root.Config, string],
	contextName cli.Destination[root.Config, string],
	namespace cli.Destination[root.Config, string],
	services Services,
) cli.Command[commandConfig] {
	return cli.Command[commandConfig]{
		Name:        "get",
		Description: cli.Description{Short: "Display one or many resources"},
		Declare: func(cb *cli.CommandBuilder[commandConfig]) {
			cb.Access.Stdin(&cb.Config.Stdin)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			client := cb.Values.Derive(
				&cb.Config.Client,
				cli.Value.ComputedBy(func(_ context.Context, _ *commandConfig, dependencies cli.Dependencies) (ClientSelection, error) {
					kubeconfigValue, ok := dependencies.Get(kubeconfig)
					if !ok {
						return ClientSelection{}, fmt.Errorf("kubeconfig dependency is unavailable")
					}
					contextValue, ok := dependencies.Get(contextName)
					if !ok {
						return ClientSelection{}, fmt.Errorf("context dependency is unavailable")
					}
					namespaceValue, ok := dependencies.Get(namespace)
					if !ok {
						return ClientSelection{}, fmt.Errorf("namespace dependency is unavailable")
					}
					return ClientSelection{
						Kubeconfig: kubeconfigValue,
						Context:    contextValue,
						Namespace:  namespaceValue,
					}, nil
				}).From(kubeconfig, contextName, namespace),
			)

			resourceType := cb.Values.Value(&cb.Config.ResourceType, resourceTypeLanguage())
			names := cb.Values.Sequence(&cb.Config.Names, cli.Value.String())
			qualifiedResources := cb.Values.Sequence(&cb.Config.QualifiedResources, qualifiedResourceLanguage())

			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[commandConfig]) {
				form.ValueFor(resourceType, "TYPE[.VERSION][.GROUP]").
					CompleteBy(completeResourceType(services)).
					From(client)
				form.Sequence(names, "NAME", cli.Count.ZeroOrMore()).
					CompleteBy(completeResourceName(services, resourceType)).
					From(client, resourceType.OptionalDependency())
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[commandConfig]) {
				form.Sequence(qualifiedResources, "TYPE[.VERSION][.GROUP]/NAME", cli.Count.OneOrMore()).
					CompleteBy(completeQualifiedResource(services)).
					From(client)
			})
			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[commandConfig]) {})

			output := cb.Values.Value(&cb.Config.Output, outputValueLanguage())
			cb.Values.Default(output, "")
			outputFlag := cb.Flags.ValueFor(output).ValueName("FORMAT")
			outputFlag.CompleteBy(completeOutputFormat).Long("output")
			outputFlag.Short('o')

			noHeaders := cb.Flags.Bool(&cb.Config.NoHeaders)
			noHeaders.Long("no-headers")

			showManagedFields := cb.Flags.Bool(&cb.Config.ShowManagedFields)
			showManagedFields.Long("show-managed-fields")

			template := cb.Flags.String(&cb.Config.Template).ValueName("TEMPLATE")
			template.Long("template")

			allowMissingTemplateKeys := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissingTemplateKeys, true)
			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(allowMissingTemplateKeys).ValueName("BOOL")
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			showLabels := cb.Flags.Bool(&cb.Config.ShowLabels)
			showLabels.Long("show-labels")

			sortBy := cb.Flags.String(&cb.Config.SortBy).ValueName("JSONPATH")
			sortBy.Long("sort-by")

			labelColumns := cb.Values.Sequence(&cb.Config.LabelColumns, cli.Value.String())
			labelColumnsFlag := cb.Flags.Sequence(labelColumns).
				ValuesPerOccurrence(cli.Count.OneOrMore()).
				DelimitedBy(",").
				ValueName("LABEL")
			labelColumnsFlag.Long("label-columns")
			labelColumnsFlag.Short('L')

			showKind := cb.Flags.Bool(&cb.Config.ShowKind)
			showKind.Long("show-kind")

			raw := cb.Flags.String(&cb.Config.Raw).ValueName("URI")
			raw.Long("raw")

			watch := cb.Flags.Bool(&cb.Config.Watch)
			watch.Long("watch")
			watch.Short('w')

			watchOnly := cb.Flags.Bool(&cb.Config.WatchOnly)
			watchOnly.Long("watch-only")

			outputWatchEvents := cb.Flags.Bool(&cb.Config.OutputWatchEvents)
			outputWatchEvents.Long("output-watch-events")

			ignoreNotFound := cb.Flags.Bool(&cb.Config.IgnoreNotFound)
			ignoreNotFound.Long("ignore-not-found")

			fieldSelector := cb.Flags.String(&cb.Config.FieldSelector).ValueName("SELECTOR")
			fieldSelector.Long("field-selector")

			allNamespaces := cb.Flags.Bool(&cb.Config.AllNamespaces)
			allNamespaces.Long("all-namespaces")
			allNamespaces.Short('A')

			serverPrint := cb.Values.Bool(&cb.Config.ServerPrint)
			cb.Values.Default(serverPrint, true)
			serverPrintFlag := cb.Flags.ValueFor(serverPrint).ValueName("BOOL")
			serverPrintFlag.Long("server-print").ImplicitValue(true)

			filenames := cb.Values.Sequence(&cb.Config.Filenames, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenames).
				ValuesPerOccurrence(cli.Count.OneOrMore()).
				DelimitedBy(",").
				ValueName("FILENAME")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomize := cb.Flags.String(&cb.Config.Kustomize).ValueName("DIR")
			kustomize.Long("kustomize")
			kustomize.Short('k')

			recursive := cb.Flags.Bool(&cb.Config.Recursive)
			recursive.Long("recursive")
			recursive.Short('R')

			chunkSize := cb.Values.Value(&cb.Config.ChunkSize, int64Language())
			cb.Values.Default(chunkSize, 500)
			chunkSizeFlag := cb.Flags.ValueFor(chunkSize).ValueName("SIZE")
			chunkSizeFlag.Long("chunk-size")

			labelSelector := cb.Flags.String(&cb.Config.LabelSelector).ValueName("SELECTOR")
			labelSelector.Long("selector")
			labelSelector.Short('l')

			subresource := cb.Flags.String(&cb.Config.Subresource).ValueName("SUBRESOURCE")
			subresource.Long("subresource")

			rawSelected := cb.When.NotEqual(raw.Destination(), "")
			cb.Rule.When(rawSelected).Forbid(resourceType)
			cb.Rule.When(rawSelected).Forbid(names)
			cb.Rule.When(rawSelected).Forbid(qualifiedResources)
			cb.Rule.When(rawSelected).Forbid(watch.Destination())
			cb.Rule.When(rawSelected).Forbid(watchOnly.Destination())
			cb.Rule.When(rawSelected).Forbid(labelSelector.Destination())
			cb.Rule.When(cb.Rule.All(rawSelected, cb.Rule.ContributedBy(output, cli.ValueProducerKindFlag))).Forbid(output)

			cb.Rule.When(cb.When.Equal(showLabels.Destination(), true)).In(output, "", "wide")
			cb.Rule.When(cb.When.Equal(outputWatchEvents.Destination(), true)).AtLeastOne(watch.Destination(), watchOnly.Destination())
			cb.Rule.AtMostOne(filenames, kustomize.Destination())
			cb.Rule.When(cb.When.Equal(recursive.Destination(), true)).Forbid(kustomize.Destination())
			cb.Rule.AtLeastOne(resourceType, qualifiedResources).
				And(filenames).
				And(kustomize.Destination()).
				And(raw.Destination())

			cb.Rule.Check("kubectl-get-raw-uri", func(_ context.Context, cfg *commandConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" {
					return true, nil
				}
				_, err := url.ParseRequestURI(cfg.Raw)
				return err == nil, nil
			}).DependsOn(raw.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-label-columns-output", func(_ context.Context, cfg *commandConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.LabelColumns) == 0 || !strings.HasPrefix(cfg.Output, "custom-columns"), nil
			}).
				DependsOn(labelColumns.OptionalDependency()).
				DependsOn(output.OptionalDependency())
		},
		Run: func(ctx context.Context, cfg *commandConfig) error {
			return run(ctx, cfg, services)
		},
	}
}

package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
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
	language := cli.ValueCodec[string]{
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
	return language
}

func qualifiedResourceLanguage() cli.ValueCodec[string] {
	language := cli.ValueCodec[string]{
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
	return language
}

func outputValueLanguage() cli.ValueCodec[string] {
	language := cli.ValueCodec[string]{
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
	return language
}

func completeOutputFormat(_ context.Context, _ *getConfig, _ cli.ValueCompletionRequest) (cli.CompletionContribution, error) {
	candidates := make([]cli.CompletionCandidate, 0, len(outputFormats))
	for _, format := range outputFormats {
		candidates = append(candidates, cli.CompletionCandidate{
			Value: format,
			Kind:  cli.CompletionCandidateValue,
		})
	}
	completion := cli.CompletionContribution{
		Candidates: candidates,
	}
	return completion, nil
}

func getCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[getConfig] {
	cmd := cli.Command[getConfig]{
		Name: "get",
		Description: cli.Description{
			Short: "Display one or many resources",
			Long:  "Display one or many resources.\n\n Prints a table of the most important information about the specified resources. You can filter the list using a label selector and the --selector flag. If the desired resource type is namespaced you will only see results in the current namespace if you don't specify any namespace.\n\n By specifying the output as 'template' and providing a Go template as the value of the --template flag, you can filter the attributes of the fetched resources.\n\nUse \"kubectl api-resources\" for a complete list of supported resources.",
		},
		Examples: []cli.Example{
			{
				Description: "List all pods in ps output format",
				CommandLine: "kubectl get pods",
			},
			{
				Description: "List all pods in ps output format with more information (such as node name)",
				CommandLine: "kubectl get pods -o wide",
			},
			{
				Description: "List a single replication controller with specified NAME in ps output format",
				CommandLine: "kubectl get replicationcontroller web",
			},
			{
				Description: "List deployments in JSON output format, in the \"v1\" version of the \"apps\" API group",
				CommandLine: "kubectl get deployments.v1.apps -o json",
			},
			{
				Description: "List a single pod in JSON output format",
				CommandLine: "kubectl get -o json pod web-pod-13je7",
			},
			{
				Description: "List a pod identified by type and name specified in \"pod.yaml\" in JSON output format",
				CommandLine: "kubectl get -f pod.yaml -o json",
			},
			{
				Description: "List resources from a directory with kustomization.yaml - e.g. dir/kustomization.yaml",
				CommandLine: "kubectl get -k dir/",
			},
			{
				Description: "Return only the phase value of the specified pod",
				CommandLine: "kubectl get -o template pod/web-pod-13je7 --template={{.status.phase}}",
			},
			{
				Description: "List resource information in custom columns",
				CommandLine: "kubectl get pod test-pod -o custom-columns=CONTAINER:.spec.containers[0].name,IMAGE:.spec.containers[0].image",
			},
			{
				Description: "List all replication controllers and services together in ps output format",
				CommandLine: "kubectl get rc,services",
			},
			{
				Description: "List one or more resources by their type and names",
				CommandLine: "kubectl get rc/web service/frontend pods/web-pod-13je7",
			},
			{
				Description: "List the 'status' subresource for a single pod",
				CommandLine: "kubectl get pod web-pod-13je7 --subresource status",
			},
			{
				Description: "List all deployments in namespace 'backend'",
				CommandLine: "kubectl get deployments.apps --namespace backend",
			},
			{
				Description: "List all pods existing in all namespaces",
				CommandLine: "kubectl get pods --all-namespaces",
			},
		},
		Declare: func(cb *cli.CommandBuilder[getConfig]) {
			cb.Access.Stdin(&cb.Config.Stdin)
			cb.Access.Stdout(&cb.Config.Stdout)
			cb.Access.Stderr(&cb.Config.Stderr)

			client := deriveClientConfig(cb, &cb.Config.Client, client)

			resourceType := cb.Values.Value(&cb.Config.ResourceType, resourceTypeLanguage())
			names := cb.Values.Sequence(&cb.Config.Names, cli.Value.String())
			qualifiedResources := cb.Values.Sequence(&cb.Config.QualifiedResources, qualifiedResourceLanguage())

			cb.Arguments.Form(func(_ *cli.ArgumentFormBuilder[getConfig]) {})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[getConfig]) {
				form.ValueFor(resourceType, "TYPE[.VERSION][.GROUP]").
					CompleteBy(completeResourceType).
					From(client)
				form.Sequence(names, "NAME", cli.Count.ZeroOrMore()).
					CompleteBy(completeResourceName(resourceType)).
					From(client, resourceType.OptionalDependency())
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[getConfig]) {
				form.Sequence(qualifiedResources, "TYPE[.VERSION][.GROUP]/NAME", cli.Count.OneOrMore()).
					CompleteBy(completeQualifiedResource).
					From(client)
			})
			output := cb.Values.Value(&cb.Config.Output, outputValueLanguage())
			cb.Values.Default(output, "")
			outputFlag := cb.Flags.ValueFor(output).ValueName("FORMAT")
			outputFlag.CompleteBy(completeOutputFormat).Long("output")
			outputFlag.Short('o')

			noHeaders := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders))
			noHeaders.Long("no-headers").ImplicitValue(true)

			showManagedFields := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFields.Long("show-managed-fields").ImplicitValue(true)

			template := cb.Flags.String(&cb.Config.Template).ValueName("TEMPLATE")
			template.Long("template")

			allowMissingTemplateKeys := cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys)
			cb.Values.Default(allowMissingTemplateKeys, true)
			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(allowMissingTemplateKeys).ValueName("BOOL")
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			showLabels := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowLabels))
			showLabels.Long("show-labels").ImplicitValue(true)

			sortBy := cb.Flags.String(&cb.Config.SortBy).ValueName("JSONPATH")
			sortBy.Long("sort-by")

			labelColumns := cb.Values.Sequence(&cb.Config.LabelColumns, cli.Value.String())
			labelColumnsFlag := cb.Flags.Sequence(labelColumns).
				ValuesPerOccurrence(cli.Count.OneOrMore()).
				DelimitedBy(",").
				ValueName("LABEL")
			labelColumnsFlag.Long("label-columns")
			labelColumnsFlag.Short('L')

			showKind := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowKind))
			showKind.Long("show-kind").ImplicitValue(true)

			raw := cb.Flags.String(&cb.Config.Raw).ValueName("URI")
			raw.Long("raw")

			watch := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Watch))
			watch.Long("watch").ImplicitValue(true)
			watch.Short('w').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			watchOnly := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.WatchOnly))
			watchOnly.Long("watch-only").ImplicitValue(true)

			outputWatchEvents := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.OutputWatchEvents))
			outputWatchEvents.Long("output-watch-events").ImplicitValue(true)

			ignoreNotFound := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.IgnoreNotFound))
			ignoreNotFound.Long("ignore-not-found").ImplicitValue(true)

			fieldSelector := cb.Flags.String(&cb.Config.FieldSelector).ValueName("SELECTOR")
			fieldSelector.Long("field-selector")

			allNamespaces := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespaces.Long("all-namespaces").ImplicitValue(true)
			allNamespaces.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

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

			recursive := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursive.Long("recursive").ImplicitValue(true)
			recursive.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			chunkSize := cb.Values.Value(&cb.Config.ChunkSize, int64Language())
			cb.Values.Default(chunkSize, 500)
			chunkSizeFlag := cb.Flags.ValueFor(chunkSize).ValueName("SIZE")
			chunkSizeFlag.Long("chunk-size")

			labelSelector := cb.Flags.String(&cb.Config.LabelSelector).ValueName("SELECTOR")
			labelSelector.Long("selector")
			labelSelector.Short('l')

			subresource := cb.Flags.String(&cb.Config.Subresource).ValueName("SUBRESOURCE")
			subresource.CompleteBy(literalCompleter[getConfig](
				cli.CompletionCandidateValue,
				"status", "scale", "resize",
			))
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

			cb.Rule.Check("kubectl-get-raw-uri", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" {
					return true, nil
				}
				_, err := url.ParseRequestURI(cfg.Raw)
				return err == nil, nil
			}).DependsOn(raw.Destination().OptionalDependency())

			cb.Rule.Check("kubectl-get-label-columns-output", func(_ context.Context, cfg *getConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.LabelColumns) == 0 || !strings.HasPrefix(cfg.Output, "custom-columns"), nil
			}).
				DependsOn(labelColumns.OptionalDependency()).
				DependsOn(output.OptionalDependency())
		},
		Run: runGet,
	}
	return cmd
}

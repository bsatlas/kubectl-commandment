package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type eventsConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	AllNamespaces            bool
	AllowMissingTemplateKeys bool
	ChunkSize                int64
	For                      string
	NoHeaders                bool
	Output                   string
	ShowManagedFields        bool
	Template                 string
	Types                    []string
	Watch                    bool
}

func eventsCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[eventsConfig] {
	cmd := cli.Command[eventsConfig]{
		Name: "events",
		Description: cli.Description{
			Short: "List events",
			Long:  "Display events.\n\n Prints a table of the most important information about events. You can request events for a namespace, for all namespace, or filtered to only those pertaining to a specified resource.",
		},
		Examples: []cli.Example{
			{
				Description: "List recent events in the default namespace",
				CommandLine: "kubectl events",
			},
			{
				Description: "List recent events in all namespaces",
				CommandLine: "kubectl events --all-namespaces",
			},
			{
				Description: "List recent events for the specified pod, then wait for more events and list them as they arrive",
				CommandLine: "kubectl events --for pod/web-pod-13je7 --watch",
			},
			{
				Description: "List recent events in YAML format",
				CommandLine: "kubectl events -oyaml",
			},
			{
				Description: "List recent only events of type 'Warning' or 'Normal'",
				CommandLine: "kubectl events --types=Warning,Normal",
			},
		},
		Declare: func(cb *cli.CommandBuilder[eventsConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.").Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.").Long("allow-missing-template-keys").ImplicitValue(true)
			cb.Values.Default(allowMissingTemplateKeysFlag, true)

			chunkSizeFlag := cb.Flags.Value(&cb.Config.ChunkSize, int64Language())
			chunkSizeFlag.Description("Return large lists in chunks rather than all at once. Pass 0 to disable.").Long("chunk-size")
			cb.Values.Default(chunkSizeFlag, int64(500))

			forFlag := cb.Flags.String(&cb.Config.For)
			forFlag.Description("Filter events to only those pertaining to the specified resource.").Long("for")

			noHeadersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders))
			noHeadersFlag.Description("When using the default output format, don't print headers.").Long("no-headers").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).").Long("output")
			outputFlag.Short('o')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.").Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file.").Long("template")

			typesValue := cb.Values.Sequence(&cb.Config.Types, cli.Value.String())
			typesFlag := cb.Flags.Sequence(typesValue)
			typesFlag.DelimitedBy(",").Description("Output only events of given types.").Long("types")

			watchFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Watch))
			watchFlag.Description("After listing the requested events, watch for more events.").Long("watch").ImplicitValue(true)
			watchFlag.Short('w').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
		},
		Run: runEvents,
	}
	return cmd
}

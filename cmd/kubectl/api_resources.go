package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type apiResourcesConfig struct {
	Client            clientConfig
	Stdout            io.Writer
	APIGroupState     cli.OptionalValue[cli.ValueState]
	NamespacedState   cli.OptionalValue[cli.ValueState]
	APIGroup          string
	Cached            bool
	Categories        []string
	Namespaced        bool
	NoHeaders         bool
	Output            string
	ShowManagedFields bool
	SortBy            string
	Verbs             []string
}

func apiResourcesCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[apiResourcesConfig] {
	cmd := cli.Command[apiResourcesConfig]{
		Name: "api-resources",
		Description: cli.Description{
			Short: "Print the supported API resources on the server",
			Long:  "Print the supported API resources on the server.",
		},
		Examples: []cli.Example{
			{
				Description: "Print the supported API resources",
				CommandLine: "kubectl api-resources",
			},
			{
				Description: "Print the supported API resources with more information",
				CommandLine: "kubectl api-resources -o wide",
			},
			{
				Description: "Print the supported API resources sorted by a column",
				CommandLine: "kubectl api-resources --sort-by=name",
			},
			{
				Description: "Print the supported namespaced resources",
				CommandLine: "kubectl api-resources --namespaced=true",
			},
			{
				Description: "Print the supported non-namespaced resources",
				CommandLine: "kubectl api-resources --namespaced=false",
			},
			{
				Description: "Print the supported API resources with a specific APIGroup",
				CommandLine: "kubectl api-resources --api-group=rbac.authorization.k8s.io",
			},
		},
		Declare: func(cb *cli.CommandBuilder[apiResourcesConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Stdout)

			apiGroupFlag := cb.Flags.String(&cb.Config.APIGroup)
			apiGroupFlag = apiGroupFlag.Description("Limit to resources in the specified API group.")
			apiGroupFlag.Long("api-group")
			cb.Access.ValueState(apiGroupFlag, &cb.Config.APIGroupState)

			cachedFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Cached))
			cachedFlag = cachedFlag.Description("Use the cached list of resources if available.")
			cachedFlag.Long("cached").ImplicitValue(true)

			categoriesValue := cb.Values.Sequence(&cb.Config.Categories, cli.Value.String())
			categoriesFlag := cb.Flags.Sequence(categoriesValue)
			categoriesFlag = categoriesFlag.DelimitedBy(",")
			categoriesFlag = categoriesFlag.Description("Limit to resources that belong to the specified categories.")
			categoriesFlag.Long("categories")

			namespacedFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Namespaced))
			namespacedFlag = namespacedFlag.Description("If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default.")
			cb.Values.Default(namespacedFlag, true)
			namespacedFlag.Long("namespaced").ImplicitValue(true)
			cb.Access.ValueState(namespacedFlag, &cb.Config.NamespacedState)

			noHeadersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders))
			noHeadersFlag = noHeadersFlag.Description("When using the default or custom-column output format, don't print headers (default print headers).")
			noHeadersFlag.Long("no-headers").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, wide).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			sortByFlag := cb.Flags.String(&cb.Config.SortBy)
			sortByFlag = sortByFlag.Description("If non-empty, sort list of resources using specified field. The field can be either 'name' or 'kind'.")
			sortByFlag.Long("sort-by")

			verbsValue := cb.Values.Sequence(&cb.Config.Verbs, cli.Value.String())
			verbsFlag := cb.Flags.Sequence(verbsValue)
			verbsFlag = verbsFlag.DelimitedBy(",")
			verbsFlag = verbsFlag.Description("Limit to resources that support the specified verbs.")
			verbsFlag.Long("verbs")
		},
		Run: runAPIResources,
	}
	return cmd
}

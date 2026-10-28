package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type explainConfig struct {
	Client     clientConfig
	Streams    genericiooptions.IOStreams
	Resource   string
	APIVersion string
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

			apiVersion := cb.Values.String(&cb.Config.APIVersion)
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

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag.Description("Format in which to render the schema (plaintext, plaintext-openapiv2)").Long("output")
			outputFlag.Short('o')
			cb.Values.Default(outputFlag, "plaintext")

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag.Description("Print the fields of fields. Use --max-depth to cap the recursion depth.").Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
		},
		Run: runExplain,
	}
	return cmd
}

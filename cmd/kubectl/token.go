package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type createTokenConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	TrailingArguments        []string
	AllowMissingTemplateKeys bool
	Audiences                []string
	BoundObjectKind          string
	BoundObjectName          string
	BoundObjectUID           string
	Duration                 time.Duration
	Output                   string
	ShowManagedFields        bool
	Template                 string
}

func createTokenCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[createTokenConfig] {
	cmd := cli.Command[createTokenConfig]{
		Name: "token",
		Description: cli.Description{
			Short: "Request a service account token",
			Long:  "Request a service account token.",
		},
		Examples: []cli.Example{
			{
				Description: "Request a token to authenticate to the kube-apiserver as the service account \"myapp\" in the current namespace",
				CommandLine: "kubectl create token myapp",
			},
			{
				Description: "Request a token for a service account in a custom namespace",
				CommandLine: "kubectl create token myapp --namespace myns",
			},
			{
				Description: "Request a token with a custom expiration",
				CommandLine: "kubectl create token myapp --duration 10m",
			},
			{
				Description: "Request a token with a custom audience",
				CommandLine: "kubectl create token myapp --audience https://example.com",
			},
			{
				Description: "Request a token bound to an instance of a Secret object",
				CommandLine: "kubectl create token myapp --bound-object-kind Secret --bound-object-name mysecret",
			},
			{
				Description: "Request a token bound to an instance of a Secret object with a specific UID",
				CommandLine: "kubectl create token myapp --bound-object-kind Secret --bound-object-name mysecret --bound-object-uid 0d4691ed-659b-4935-a832-355f77ee47cc",
			},
		},
		Declare: func(cb *cli.CommandBuilder[createTokenConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			cb.Dialect.EndOfOptions()

			name := cb.Values.String(&cb.Config.Name)
			trailingArguments := cb.Values.Sequence(&cb.Config.TrailingArguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createTokenConfig]) {
				form.ValueFor(name, "NAME").
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *createTokenConfig) (clientConfig, []string) { return cfg.Client, nil },
						"serviceaccount",
					)).
					From(client)
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[createTokenConfig]) {
				form.ValueFor(name, "NAME").
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *createTokenConfig) (clientConfig, []string) { return cfg.Client, nil },
						"serviceaccount",
					)).
					From(client)
				form.Literal("--").EndOfOptions()
				form.PassThrough(trailingArguments, "ARG", cli.Count.ZeroOrMore())
			})

			allowMissingTemplateKeysFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllowMissingTemplateKeys))
			allowMissingTemplateKeysFlag = allowMissingTemplateKeysFlag.Description("If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats.")
			cb.Values.Default(allowMissingTemplateKeysFlag, true)
			allowMissingTemplateKeysFlag.Long("allow-missing-template-keys").ImplicitValue(true)

			audienceValue := cb.Values.Sequence(&cb.Config.Audiences, cli.Value.String())
			audienceFlag := cb.Flags.Sequence(audienceValue)
			audienceFlag = audienceFlag.Description("Audience of the requested token. If unset, defaults to requesting a token for use with the Kubernetes API server. May be repeated to request a token valid for multiple audiences.")
			audienceFlag.Long("audience")

			boundObjectKindFlag := cb.Flags.String(&cb.Config.BoundObjectKind)
			boundObjectKindFlag = boundObjectKindFlag.Description("Kind of an object to bind the token to. Supported kinds are Node, Pod, Secret. If set, --bound-object-name must be provided.")
			boundObjectKindFlag.Long("bound-object-kind")

			boundObjectNameFlag := cb.Flags.String(&cb.Config.BoundObjectName)
			boundObjectNameFlag = boundObjectNameFlag.Description("Name of an object to bind the token to. The token will expire when the object is deleted. Requires --bound-object-kind.")
			boundObjectNameFlag.Long("bound-object-name")

			boundObjectUIDFlag := cb.Flags.String(&cb.Config.BoundObjectUID)
			boundObjectUIDFlag = boundObjectUIDFlag.Description("UID of an object to bind the token to. Requires --bound-object-kind and --bound-object-name. If unset, the UID of the existing object is used.")
			boundObjectUIDFlag.Long("bound-object-uid")

			durationFlag := cb.Flags.Value(&cb.Config.Duration, durationLanguage())
			durationFlag = durationFlag.Description("Requested lifetime of the issued token. If not set or if set to 0, the lifetime will be determined by the server automatically. The server may return a token with a longer or shorter lifetime.")
			durationFlag.Long("duration")

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output format. One of: (json, yaml, kyaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			showManagedFieldsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShowManagedFields))
			showManagedFieldsFlag = showManagedFieldsFlag.Description("If true, keep the managedFields when printing objects in JSON or YAML format.")
			showManagedFieldsFlag.Long("show-managed-fields").ImplicitValue(true)

			templateFlag := cb.Flags.String(&cb.Config.Template)
			templateFlag = templateFlag.Description("Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].")
			templateFlag.Long("template")
		},
		Run: runCreateToken,
	}
	return cmd
}

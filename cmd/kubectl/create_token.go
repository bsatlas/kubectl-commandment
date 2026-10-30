package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/create"
	"k8s.io/kubectl/pkg/cmd/util"
)

var tokenAudienceLanguage = cli.ValueCodec[string]{
	Name: "token-audience",
	Parse: func(value string) (string, error) {
		if value == "" {
			return "", cli.Rejection.Message("token-audience", "audience must not be empty", cli.Rejection.At(0, 0, 0))
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

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
	OutputState              cli.OptionalValue[cli.ValueState]
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

			cb.Rule.NotEmpty(name)

			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)

			audienceValue := cb.Values.Sequence(&cb.Config.Audiences, tokenAudienceLanguage)
			audienceFlag := cb.Flags.Sequence(audienceValue)
			audienceFlag.Description("Audience of the requested token. If unset, defaults to requesting a token for use with the Kubernetes API server. May be repeated to request a token valid for multiple audiences.")
			audienceFlag.Long("audience")

			boundObjectKindFlag := cb.Flags.String(&cb.Config.BoundObjectKind)
			cb.Values.ResolveWith(boundObjectKindFlag, cli.Value.KeepLast())
			boundObjectKindFlag.Description("Kind of an object to bind the token to. Supported kinds are Node, Pod, Secret. If set, --bound-object-name must be provided.")
			boundObjectKindFlag.Long("bound-object-kind")

			boundObjectNameFlag := cb.Flags.String(&cb.Config.BoundObjectName)
			cb.Values.ResolveWith(boundObjectNameFlag, cli.Value.KeepLast())
			boundObjectNameFlag.Description("Name of an object to bind the token to. The token will expire when the object is deleted. Requires --bound-object-kind.")
			boundObjectNameFlag.Long("bound-object-name")

			boundObjectUIDFlag := cb.Flags.String(&cb.Config.BoundObjectUID)
			cb.Values.ResolveWith(boundObjectUIDFlag, cli.Value.KeepLast())
			boundObjectUIDFlag.Description("UID of an object to bind the token to. Requires --bound-object-kind and --bound-object-name. If unset, the UID of the existing object is used.")
			boundObjectUIDFlag.Long("bound-object-uid")

			durationFlag := cb.Flags.Value(&cb.Config.Duration, durationLanguage())
			cb.Values.ResolveWith(durationFlag, cli.Value.KeepLast())
			durationFlag.Description("Requested lifetime of the issued token. If not set or if set to 0, the lifetime will be determined by the server automatically. The server may return a token with a longer or shorter lifetime.")
			cb.Values.Default(durationFlag, time.Duration(0))
			durationFlag.Long("duration")

			cb.Rule.AtLeast(durationFlag.Destination(), time.Duration(0))
			cb.Rule.Check("kubectl-create-token-duration-seconds", func(ctx context.Context, cfg *createTokenConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Duration%time.Second == 0, nil
			}).DependsOn(durationFlag.Destination())
			kindSet := cb.When.NotEqual(boundObjectKindFlag.Destination(), "")
			cb.Rule.When(kindSet).In(boundObjectKindFlag.Destination(), "Node", "Pod", "Secret")
			cb.Rule.When(kindSet).Require(boundObjectNameFlag.Destination())
			cb.Rule.Unless(kindSet).Forbid(boundObjectNameFlag.Destination())
			cb.Rule.Unless(kindSet).Forbid(boundObjectUIDFlag.Destination())
		},
		Run: runCreateToken,
	}
	return cmd
}

func runCreateToken(ctx context.Context, cfg *createTokenConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	options := create.NewTokenOpts(cfg.Streams)
	options.Name = cfg.Name
	options.BoundObjectKind = cfg.BoundObjectKind
	options.BoundObjectName = cfg.BoundObjectName
	options.BoundObjectUID = cfg.BoundObjectUID
	options.Audiences = slices.Clone(cfg.Audiences)
	options.Duration = cfg.Duration

	var err error
	options.Namespace, _, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.CoreClient = clientset.CoreV1()
	configurePrintFlagsWithOutputState(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	options.PrintObj, err = printObject(options.PrintFlags, util.DryRunNone, cfg.Streams.Out)
	if err != nil {
		return err
	}
	if options.CoreClient == nil {
		return fmt.Errorf("no client provided")
	}
	if options.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	return options.Run()
}

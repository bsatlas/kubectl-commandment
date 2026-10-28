package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd/api"
)

type configPair struct {
	Name   string
	Value  string
	Remove bool
}

func parseConfigPair(name, text string) (configPair, error) {
	if pairName, pairValue, ok := strings.Cut(text, "="); ok && pairName != "" {
		pair := configPair{
			Name:  pairName,
			Value: pairValue,
		}
		return pair, nil
	}
	if pairName, ok := strings.CutSuffix(text, "-"); ok && pairName != "" {
		pair := configPair{
			Name:   pairName,
			Remove: true,
		}
		return pair, nil
	}
	return configPair{}, cli.Rejection.Message(
		cli.InputRejectionCode("invalid-"+name),
		fmt.Sprintf("invalid %s format: %s", name, text),
		cli.Rejection.At(0, 0, len(text)),
	)
}

func formatConfigPair(pair configPair) string {
	if pair.Remove {
		return pair.Name + "-"
	}
	return pair.Name + "=" + pair.Value
}

func configPairLanguage(name string) cli.ValueCodec[configPair] {
	language := cli.ValueCodec[configPair]{
		Name:     name,
		Parse:    func(text string) (configPair, error) { return parseConfigPair(name, text) },
		Format:   formatConfigPair,
		Snapshot: func(pair configPair) configPair { return pair },
	}
	return language
}

func parseConfigCSV(name, text string) ([]string, error) {
	if text == "" {
		return nil, nil
	}
	values, err := csv.NewReader(strings.NewReader(text)).Read()
	if err != nil {
		return nil, cli.Rejection.Message(
			cli.InputRejectionCode("invalid-"+name),
			fmt.Sprintf("invalid %s format: %v", name, err),
			cli.Rejection.At(0, 0, len(text)),
		)
	}
	return values, nil
}

func formatConfigCSV(values []string) string {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(values); err != nil {
		panic(err)
	}
	writer.Flush()
	return strings.TrimSuffix(buffer.String(), "\n")
}

func configPairCSVLanguage(name string) cli.TokenLanguage[[]configPair] {
	language := cli.TokenLanguage[[]configPair]{
		Name:   name,
		Count:  cli.Count.Exactly(1),
		Syntax: []string{"CSV"},
		Parse: func(_ context.Context, tokens []string) ([]configPair, error) {
			values, err := parseConfigCSV(name, tokens[0])
			if err != nil {
				return nil, err
			}
			pairs := make([]configPair, 0, len(values))
			for _, value := range values {
				pair, err := parseConfigPair(name, value)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
			return pairs, nil
		},
		Format: func(pairs []configPair) []string {
			values := make([]string, len(pairs))
			for index, pair := range pairs {
				values[index] = formatConfigPair(pair)
			}
			return []string{formatConfigCSV(values)}
		},
	}
	return language
}

func configStringCSVLanguage(name string) cli.TokenLanguage[[]string] {
	language := cli.TokenLanguage[[]string]{
		Name:   name,
		Count:  cli.Count.Exactly(1),
		Syntax: []string{"CSV"},
		Parse: func(_ context.Context, tokens []string) ([]string, error) {
			return parseConfigCSV(name, tokens[0])
		},
		Format: func(values []string) []string {
			return []string{formatConfigCSV(values)}
		},
	}
	return language
}

func configExecInteractiveModeLanguage() cli.ValueCodec[api.ExecInteractiveMode] {
	parse := func(text string) (api.ExecInteractiveMode, error) {
		mode := api.ExecInteractiveMode(text)
		switch mode {
		case api.IfAvailableExecInteractiveMode, api.NeverExecInteractiveMode, api.AlwaysExecInteractiveMode:
			return mode, nil
		default:
			err := cli.Rejection.Message(
				cli.InputRejectionCode("invalid-exec-interactive-mode"),
				"invalid interactive mode type, can be only IfAvailable, Never, Always",
				cli.Rejection.At(0, 0, len(text)),
			)
			return "", err
		}
	}
	language := cli.ValueCodec[api.ExecInteractiveMode]{
		Name:     "exec-interactive-mode",
		Parse:    parse,
		Format:   func(mode api.ExecInteractiveMode) string { return string(mode) },
		Snapshot: cli.Value.Assign[api.ExecInteractiveMode](),
	}
	return language
}

type configSetCredentialsConfig struct {
	Kubeconfig              string
	Name                    string
	AuthProvider            string
	AuthProviderArg         []configPair
	ClientCertificate       string
	ClientKey               string
	EmbedCerts              bool
	ExecAPIVersion          string
	ExecArg                 []string
	ExecCommand             string
	ExecEnv                 []configPair
	ExecInteractiveMode     api.ExecInteractiveMode
	ExecProvideClusterInfo  bool
	Password                string
	Token                   string
	Username                string
	AuthProviderState       cli.OptionalValue[cli.ValueState]
	ClientCertificateState  cli.OptionalValue[cli.ValueState]
	ClientKeyState          cli.OptionalValue[cli.ValueState]
	ExecAPIVersionState     cli.OptionalValue[cli.ValueState]
	ExecCommandState        cli.OptionalValue[cli.ValueState]
	ExecInteractiveState    cli.OptionalValue[cli.ValueState]
	ExecProvideClusterState cli.OptionalValue[cli.ValueState]
	PasswordState           cli.OptionalValue[cli.ValueState]
	TokenState              cli.OptionalValue[cli.ValueState]
	UsernameState           cli.OptionalValue[cli.ValueState]
	Stdout                  io.Writer
}

func configSetCredentialsCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configSetCredentialsConfig] {
	cmd := cli.Command[configSetCredentialsConfig]{
		Name: "set-credentials",
		Description: cli.Description{
			Short: "Set a user entry in kubeconfig",
			Long:  "Set a user entry in kubeconfig.\n\n Specifying a name that already exists will merge new fields on top of existing values.\n\n        Client-certificate flags:\n        --client-certificate=certfile --client-key=keyfile\n        \n        Bearer token flags:\n        --token=bearer_token\n        \n        Basic auth flags:\n        --username=basic_user --password=basic_password\n        \n Bearer token and basic auth are mutually exclusive.",
		},
		Examples: []cli.Example{
			{
				Description: "Set only the \"client-key\" field on the \"cluster-admin\"\nentry, without touching other values",
				CommandLine: "kubectl config set-credentials cluster-admin --client-key=~/.kube/admin.key",
			},
			{
				Description: "Set basic auth for the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --username=admin --password=uXFGweU9l35qcif",
			},
			{
				Description: "Embed client certificate data in the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --client-certificate=~/.kube/admin.crt --embed-certs=true",
			},
			{
				Description: "Enable the Google Compute Platform auth provider for the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --auth-provider=gcp",
			},
			{
				Description: "Enable the OpenID Connect auth provider for the \"cluster-admin\" entry with additional arguments",
				CommandLine: "kubectl config set-credentials cluster-admin --auth-provider=oidc --auth-provider-arg=client-id=foo --auth-provider-arg=client-secret=bar",
			},
			{
				Description: "Remove the \"client-secret\" config value for the OpenID Connect auth provider for the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --auth-provider=oidc --auth-provider-arg=client-secret-",
			},
			{
				Description: "Enable new exec auth plugin for the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --exec-command=/path/to/the/executable --exec-api-version=client.authentication.k8s.io/v1beta1",
			},
			{
				Description: "Enable new exec auth plugin for the \"cluster-admin\" entry with interactive mode",
				CommandLine: "kubectl config set-credentials cluster-admin --exec-command=/path/to/the/executable --exec-api-version=client.authentication.k8s.io/v1beta1 --exec-interactive-mode=Never",
			},
			{
				Description: "Define new exec auth plugin arguments for the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --exec-arg=arg1 --exec-arg=arg2",
			},
			{
				Description: "Create or update exec auth plugin environment variables for the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --exec-env=key1=val1 --exec-env=key2=val2",
			},
			{
				Description: "Remove exec auth plugin environment variables for the \"cluster-admin\" entry",
				CommandLine: "kubectl config set-credentials cluster-admin --exec-env=var-to-remove-",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configSetCredentialsConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.ValueFor(name, "NAME")
			cb.Rule.NotEmpty(name)

			authProvider := cb.Values.String(&cb.Config.AuthProvider)
			cb.Flags.ValueFor(authProvider).Description("Auth provider for the user entry in kubeconfig").Long("auth-provider")
			cb.Access.ValueState(authProvider, &cb.Config.AuthProviderState)

			authProviderArg := cb.Values.Sequence(&cb.Config.AuthProviderArg, configPairLanguage("auth-provider-arg"))
			cb.Flags.Tokens(authProviderArg, configPairCSVLanguage("auth-provider-arg")).Description("'key=value' arguments for the auth provider").Long("auth-provider-arg")

			clientCertificate := cb.Values.String(&cb.Config.ClientCertificate)
			cb.Flags.ValueFor(clientCertificate).
				Description("Path to client-certificate file for the user entry in kubeconfig").
				CompleteWith(cli.CompletionNativeFiles).
				Long("client-certificate")
			cb.Access.ValueState(clientCertificate, &cb.Config.ClientCertificateState)

			clientKey := cb.Values.String(&cb.Config.ClientKey)
			cb.Flags.ValueFor(clientKey).
				Description("Path to client-key file for the user entry in kubeconfig").
				CompleteWith(cli.CompletionNativeFiles).
				Long("client-key")
			cb.Access.ValueState(clientKey, &cb.Config.ClientKeyState)

			embedCerts := cb.Values.Bool(&cb.Config.EmbedCerts)
			cb.Flags.ValueFor(embedCerts).Description("Embed client cert/key for the user entry in kubeconfig").Long("embed-certs").ImplicitValue(true)

			execAPIVersion := cb.Values.String(&cb.Config.ExecAPIVersion)
			cb.Flags.ValueFor(execAPIVersion).Description("API version of the exec credential plugin for the user entry in kubeconfig").Long("exec-api-version")
			cb.Access.ValueState(execAPIVersion, &cb.Config.ExecAPIVersionState)

			execArg := cb.Values.Sequence(&cb.Config.ExecArg, cli.Value.String())
			cb.Flags.Tokens(execArg, configStringCSVLanguage("exec-arg")).Description("New arguments for the exec credential plugin command for the user entry in kubeconfig").Long("exec-arg")

			execCommand := cb.Values.String(&cb.Config.ExecCommand)
			cb.Flags.ValueFor(execCommand).Description("Command for the exec credential plugin for the user entry in kubeconfig").Long("exec-command")
			cb.Access.ValueState(execCommand, &cb.Config.ExecCommandState)

			execEnv := cb.Values.Sequence(&cb.Config.ExecEnv, configPairLanguage("exec-env"))
			cb.Flags.Sequence(execEnv).Description("'key=value' environment values for the exec credential plugin").Long("exec-env")

			execInteractiveMode := cb.Values.Value(&cb.Config.ExecInteractiveMode, configExecInteractiveModeLanguage())
			cb.Flags.ValueFor(execInteractiveMode).Description("InteractiveMode of the exec credentials plugin for the user entry in kubeconfig").Long("exec-interactive-mode")
			cb.Access.ValueState(execInteractiveMode, &cb.Config.ExecInteractiveState)

			execProvideClusterInfo := cb.Values.Bool(&cb.Config.ExecProvideClusterInfo)
			cb.Flags.ValueFor(execProvideClusterInfo).Description("ProvideClusterInfo of the exec credentials plugin for the user entry in kubeconfig").Long("exec-provide-cluster-info").ImplicitValue(true)
			cb.Access.ValueState(execProvideClusterInfo, &cb.Config.ExecProvideClusterState)

			password := cb.Values.String(&cb.Config.Password)
			cb.Flags.ValueFor(password).Description("password for the user entry in kubeconfig").Long("password").Sensitive()
			cb.Access.ValueState(password, &cb.Config.PasswordState)

			token := cb.Values.String(&cb.Config.Token)
			cb.Flags.ValueFor(token).Description("token for the user entry in kubeconfig").Long("token").Sensitive()
			cb.Access.ValueState(token, &cb.Config.TokenState)

			username := cb.Values.String(&cb.Config.Username)
			cb.Flags.ValueFor(username).Description("username for the user entry in kubeconfig").Long("username")
			cb.Access.ValueState(username, &cb.Config.UsernameState)

			cb.Rule.When(cb.When.NotEqual(token, "")).Equal(username, "")
			cb.Rule.When(cb.When.NotEqual(token, "")).Equal(password, "")
			cb.Rule.Check("kubectl-config-set-credentials-embed-certs", func(_ context.Context, cfg *configSetCredentialsConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.EmbedCerts || cfg.ClientCertificate != "" || cfg.ClientKey != "", nil
			}).
				DependsOn(embedCerts.OptionalDependency()).
				DependsOn(clientCertificate.OptionalDependency()).
				DependsOn(clientKey.OptionalDependency())
		},
		Run: runConfigSetCredentials,
	}
	return cmd
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/cmd/util"
)

type configSetCredentialsConfig struct {
	Kubeconfig              string
	Name                    string
	AuthProvider            string
	AuthProviderArg         []string
	ClientCertificate       string
	ClientKey               string
	EmbedCerts              bool
	ExecAPIVersion          string
	ExecArg                 []string
	ExecCommand             string
	ExecEnv                 []string
	ExecInteractiveMode     string
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

			authProvider := cb.Values.String(&cb.Config.AuthProvider)
			cb.Flags.ValueFor(authProvider).Description("Auth provider for the user entry in kubeconfig").Long("auth-provider")
			cb.Access.ValueState(authProvider, &cb.Config.AuthProviderState)

			authProviderArg := cb.Values.Sequence(&cb.Config.AuthProviderArg, cli.Value.String())
			cb.Flags.Sequence(authProviderArg).DelimitedBy(",").Description("'key=value' arguments for the auth provider").Long("auth-provider-arg")

			clientCertificate := cb.Values.String(&cb.Config.ClientCertificate)
			cb.Flags.ValueFor(clientCertificate).Description("Path to client-certificate file for the user entry in kubeconfig").Long("client-certificate")
			cb.Access.ValueState(clientCertificate, &cb.Config.ClientCertificateState)

			clientKey := cb.Values.String(&cb.Config.ClientKey)
			cb.Flags.ValueFor(clientKey).Description("Path to client-key file for the user entry in kubeconfig").Long("client-key")
			cb.Access.ValueState(clientKey, &cb.Config.ClientKeyState)

			embedCerts := cb.Values.Bool(&cb.Config.EmbedCerts)
			cb.Flags.ValueFor(embedCerts).Description("Embed client cert/key for the user entry in kubeconfig").Long("embed-certs").ImplicitValue(true)

			execAPIVersion := cb.Values.String(&cb.Config.ExecAPIVersion)
			cb.Flags.ValueFor(execAPIVersion).Description("API version of the exec credential plugin for the user entry in kubeconfig").Long("exec-api-version")
			cb.Access.ValueState(execAPIVersion, &cb.Config.ExecAPIVersionState)

			execArg := cb.Values.Sequence(&cb.Config.ExecArg, cli.Value.String())
			cb.Flags.Sequence(execArg).DelimitedBy(",").Description("New arguments for the exec credential plugin command for the user entry in kubeconfig").Long("exec-arg")

			execCommand := cb.Values.String(&cb.Config.ExecCommand)
			cb.Flags.ValueFor(execCommand).Description("Command for the exec credential plugin for the user entry in kubeconfig").Long("exec-command")
			cb.Access.ValueState(execCommand, &cb.Config.ExecCommandState)

			execEnv := cb.Values.Sequence(&cb.Config.ExecEnv, cli.Value.String())
			cb.Flags.Sequence(execEnv).Description("'key=value' environment values for the exec credential plugin").Long("exec-env")

			execInteractiveMode := cb.Values.String(&cb.Config.ExecInteractiveMode)
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
		},
		Run: runConfigSetCredentials,
	}
	return cmd
}

func runConfigSetCredentials(_ context.Context, cfg *configSetCredentialsConfig) error {
	if cfg.Name == "" {
		return errors.New("you must specify a non-empty user name")
	}
	methods := make([]string, 0, 2)
	if cfg.Token != "" {
		methods = append(methods, "--token")
	}
	if cfg.Username != "" || cfg.Password != "" {
		methods = append(methods, "--username/--password")
	}
	if len(methods) > 1 {
		return fmt.Errorf("you cannot specify more than one authentication method at the same time: %v", strings.Join(methods, ", "))
	}
	if cfg.EmbedCerts {
		if cfg.ClientCertificate == "" && cfg.ClientKey == "" {
			return fmt.Errorf("you must specify a --%s or --%s to embed", clientcmd.FlagCertFile, clientcmd.FlagKeyFile)
		}
		if cfg.ClientCertificate != "" {
			if _, err := os.Stat(cfg.ClientCertificate); err != nil {
				return fmt.Errorf("could not stat %s file %s: %v", clientcmd.FlagCertFile, cfg.ClientCertificate, err)
			}
		}
		if cfg.ClientKey != "" {
			if _, err := os.Stat(cfg.ClientKey); err != nil {
				return fmt.Errorf("could not stat %s file %s: %v", clientcmd.FlagKeyFile, cfg.ClientKey, err)
			}
		}
	}
	if flagProvided(cfg.ExecInteractiveState) {
		mode := api.ExecInteractiveMode(cfg.ExecInteractiveMode)
		if mode != api.IfAvailableExecInteractiveMode && mode != api.AlwaysExecInteractiveMode && mode != api.NeverExecInteractiveMode {
			return errors.New("invalid interactive mode type, can be only IfAvailable, Never, Always")
		}
	}

	authProviderArgs, authProviderRemovals, err := util.ParsePairs(cfg.AuthProviderArg, "auth-provider-arg", true)
	if err != nil {
		return err
	}
	execEnv, execEnvRemovals, err := util.ParsePairs(cfg.ExecEnv, "exec-env", true)
	if err != nil {
		return err
	}

	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	starting, exists := config.AuthInfos[cfg.Name]
	if !exists {
		starting = api.NewAuthInfo()
	}
	modified, err := modifyConfigAuthInfo(*starting, cfg, authProviderArgs, authProviderRemovals, execEnv, execEnvRemovals)
	if err != nil {
		return err
	}
	config.AuthInfos[cfg.Name] = &modified
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "User %q set.\n", cfg.Name)
	return err
}

func modifyConfigAuthInfo(existing api.AuthInfo, cfg *configSetCredentialsConfig, authProviderArgs map[string]string, authProviderRemovals []string, execEnv map[string]string, execEnvRemovals []string) (api.AuthInfo, error) {
	modified := existing

	if flagProvided(cfg.ClientCertificateState) {
		path := cfg.ClientCertificate
		if cfg.EmbedCerts {
			data, err := os.ReadFile(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientCertificateData = data
			modified.ClientCertificate = ""
		} else {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientCertificate = absolute
			if absolute != "" {
				modified.ClientCertificateData = nil
			}
		}
	}
	if flagProvided(cfg.ClientKeyState) {
		path := cfg.ClientKey
		if cfg.EmbedCerts {
			data, err := os.ReadFile(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientKeyData = data
			modified.ClientKey = ""
		} else {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return api.AuthInfo{}, err
			}
			modified.ClientKey = absolute
			if absolute != "" {
				modified.ClientKeyData = nil
			}
		}
	}

	setToken := false
	setBasic := false
	if flagProvided(cfg.TokenState) {
		modified.Token = cfg.Token
		setToken = modified.Token != ""
	}
	if flagProvided(cfg.UsernameState) {
		modified.Username = cfg.Username
		setBasic = modified.Username != ""
	}
	if flagProvided(cfg.PasswordState) {
		modified.Password = cfg.Password
		setBasic = setBasic || modified.Password != ""
	}

	if flagProvided(cfg.AuthProviderState) {
		if modified.AuthProvider == nil || modified.AuthProvider.Name != cfg.AuthProvider {
			modified.AuthProvider = &api.AuthProviderConfig{
				Name: cfg.AuthProvider,
			}
		}
	}
	if modified.AuthProvider != nil {
		if modified.AuthProvider.Config == nil {
			modified.AuthProvider.Config = make(map[string]string)
		}
		for _, key := range authProviderRemovals {
			delete(modified.AuthProvider.Config, key)
		}
		for key, value := range authProviderArgs {
			modified.AuthProvider.Config[key] = value
		}
	}

	if flagProvided(cfg.ExecCommandState) {
		if modified.Exec == nil {
			modified.Exec = &api.ExecConfig{
				Command: cfg.ExecCommand,
			}
		} else {
			modified.Exec.Command = cfg.ExecCommand
			modified.Exec.Args = nil
		}
	}
	if modified.Exec != nil {
		if flagProvided(cfg.ExecAPIVersionState) {
			modified.Exec.APIVersion = cfg.ExecAPIVersion
		}
		if len(cfg.ExecArg) > 0 {
			modified.Exec.Args = append([]string(nil), cfg.ExecArg...)
		}
		if flagProvided(cfg.ExecInteractiveState) {
			modified.Exec.InteractiveMode = api.ExecInteractiveMode(cfg.ExecInteractiveMode)
		}
		if flagProvided(cfg.ExecProvideClusterState) {
			modified.Exec.ProvideClusterInfo = cfg.ExecProvideClusterInfo
		}
		if len(execEnvRemovals) > 0 {
			remove := make(map[string]struct{}, len(execEnvRemovals))
			for _, name := range execEnvRemovals {
				remove[name] = struct{}{}
			}
			kept := modified.Exec.Env[:0]
			for _, variable := range modified.Exec.Env {
				if _, found := remove[variable.Name]; !found {
					kept = append(kept, variable)
				}
			}
			modified.Exec.Env = kept
		}
		for name, value := range execEnv {
			updated := false
			for index := range modified.Exec.Env {
				if modified.Exec.Env[index].Name == name {
					modified.Exec.Env[index].Value = value
					updated = true
					break
				}
			}
			if !updated {
				modified.Exec.Env = append(modified.Exec.Env, api.ExecEnvVar{
					Name:  name,
					Value: value,
				})
			}
		}
	}

	if setToken || setBasic {
		if !setToken {
			modified.Token = ""
		}
		if !setBasic {
			modified.Username = ""
			modified.Password = ""
		}
	}
	return modified, nil
}

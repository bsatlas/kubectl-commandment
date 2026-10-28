package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestConfigSetCredentialsDeclaresCredentialValueLanguages(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configSetCredentialsCommand(destinations.Kubeconfig))
	})
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "auth provider pair",
			args: []string{"config", "set-credentials", "user", "--auth-provider-arg=client-id=example"},
		},
		{
			name: "auth provider removal",
			args: []string{"config", "set-credentials", "user", "--auth-provider-arg=client-secret-"},
		},
		{
			name:     "malformed auth provider pair",
			args:     []string{"config", "set-credentials", "user", "--auth-provider-arg=missing-separator"},
			rejected: true,
		},
		{
			name: "exec environment pair",
			args: []string{"config", "set-credentials", "user", "--exec-env=TOKEN=value"},
		},
		{
			name: "empty exec environment value",
			args: []string{"config", "set-credentials", "user", "--exec-env=TOKEN="},
		},
		{
			name: "exec environment removal",
			args: []string{"config", "set-credentials", "user", "--exec-env=TOKEN-"},
		},
		{
			name:     "malformed exec environment",
			args:     []string{"config", "set-credentials", "user", "--exec-env=missing-separator"},
			rejected: true,
		},
		{
			name: "interactive mode",
			args: []string{"config", "set-credentials", "user", "--exec-interactive-mode=Never"},
		},
		{
			name:     "invalid interactive mode",
			args:     []string{"config", "set-credentials", "user", "--exec-interactive-mode=Sometimes"},
			rejected: true,
		},
		{
			name: "explicit false embed without certificate files",
			args: []string{"config", "set-credentials", "user", "--embed-certs=false"},
		},
		{
			name:     "embed requires certificate or key",
			args:     []string{"config", "set-credentials", "user", "--embed-certs"},
			rejected: true,
		},
		{
			name:     "token conflicts with username",
			args:     []string{"config", "set-credentials", "user", "--token=token", "--username=user"},
			rejected: true,
		},
		{
			name:     "token conflicts with password",
			args:     []string{"config", "set-credentials", "user", "--token=token", "--password=secret"},
			rejected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if !test.rejected {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestConfigSetCredentialsCertificateFlagsUseNativeFileCompletion(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configSetCredentialsCommand(destinations.Kubeconfig))
	})
	client := cli.CompletionClient{
		Capabilities: cli.CompletionClientCapabilities{
			Native: cli.CompletionNativeCapabilities{
				Files: true,
			},
		},
	}
	for _, flag := range []string{"--client-certificate", "--client-key"} {
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
			request := cli.CompletionRequest{
				Args: []string{"config", "set-credentials", "user", flag, ""},
				Cursor: cli.CompletionCursor{
					Argument: 4,
				},
				Client: client,
			}
			clitest.Complete(t.Context(), app, request).Require(t,
				clitest.CompletionNativeRequestsExactly(clitest.CompletionNativeRequestAt(0)),
				clitest.CompletionNativeRequestAt(0).Kind(cli.CompletionNativeFiles),
			)
		})
	}
}

func TestConfigSetCredentialsPreservesStringSliceCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		read func(string) string
		want string
	}{
		{
			name: "auth provider quoted comma",
			args: []string{"--auth-provider=oidc", `--auth-provider-arg="client-id=a,b"`},
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				return config.AuthInfos["user"].AuthProvider.Config["client-id"]
			},
			want: "a,b",
		},
		{
			name: "exec argument quoted comma",
			args: []string{"--exec-command=plugin", `--exec-arg="a,b"`},
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				arguments := config.AuthInfos["user"].Exec.Args
				if len(arguments) != 1 {
					return "unexpected argument count"
				}
				return arguments[0]
			},
			want: "a,b",
		},
		{
			name: "exec argument repeated CSV",
			args: []string{"--exec-command=plugin", "--exec-arg=a,b", "--exec-arg=c"},
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				arguments := config.AuthInfos["user"].Exec.Args
				if len(arguments) != 3 {
					return "unexpected argument count"
				}
				return arguments[0] + "," + arguments[1] + "," + arguments[2]
			},
			want: "a,b,c",
		},
		{
			name: "exec environment does not use CSV",
			args: []string{"--exec-command=plugin", "--exec-env=TOKEN=a,b"},
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				environment := config.AuthInfos["user"].Exec.Env
				if len(environment) != 1 {
					return "unexpected environment count"
				}
				return environment[0].Name + "=" + environment[0].Value
			},
			want: "TOKEN=a,b",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := writeConfigTestKubeconfig(t, "dev", "dev")
			app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
				config.Subcommands.Command(configSetCredentialsCommand(destinations.Kubeconfig))
			})
			args := []string{"--kubeconfig", path, "config", "set-credentials", "user"}
			args = append(args, test.args...)
			request := cli.RunRequest{
				Args:   args,
				Stdout: &bytes.Buffer{},
			}
			if err := app.Run(t.Context(), request); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if got := test.read(path); got != test.want {
				t.Fatalf("stored value = %q, want %q", got, test.want)
			}
		})
	}
}

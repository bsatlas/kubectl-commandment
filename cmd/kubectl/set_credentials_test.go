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

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
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

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
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

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	command := clitest.RootCommand().Command("config").Command("set-credentials")

	t.Run("auth provider quoted comma", func(t *testing.T) {
		t.Parallel()

		value := command.Flag("--auth-provider-arg").Value()
		candidate := value.CandidateAt(0)
		request := cli.InspectionRequest{
			Args: []string{"config", "set-credentials", "user", `--auth-provider-arg="client-id=a,b"`},
		}
		clitest.Inspect(t.Context(), app, request).Require(t,
			clitest.InspectionReady(),
			value.Resolved(),
			value.CandidatesExactly(candidate),
			candidate.ElementCount(1),
			value.ElementsExactly(candidate.ElementAt(0)),
			candidate.ElementAt(0).Text("client-id=a,b"),
		)
	})

	t.Run("exec argument quoted comma", func(t *testing.T) {
		t.Parallel()

		value := command.Flag("--exec-arg").Value()
		candidate := value.CandidateAt(0)
		request := cli.InspectionRequest{
			Args: []string{"config", "set-credentials", "user", `--exec-arg="a,b"`},
		}
		clitest.Inspect(t.Context(), app, request).Require(t,
			clitest.InspectionReady(),
			value.Resolved(),
			value.CandidatesExactly(candidate),
			candidate.ElementCount(1),
			value.ElementsExactly(candidate.ElementAt(0)),
			candidate.ElementAt(0).Text("a,b"),
		)
	})

	t.Run("exec argument repeated CSV", func(t *testing.T) {
		t.Parallel()

		value := command.Flag("--exec-arg").Value()
		first := value.CandidateAt(0)
		second := value.CandidateAt(1)
		request := cli.InspectionRequest{
			Args: []string{"config", "set-credentials", "user", "--exec-arg=a,b", "--exec-arg=c"},
		}
		clitest.Inspect(t.Context(), app, request).Require(t,
			clitest.InspectionReady(),
			value.Resolved(),
			value.CandidatesExactly(first, second),
			first.ElementCount(2),
			second.ElementCount(1),
			value.ElementsExactly(first.ElementAt(0), first.ElementAt(1), second.ElementAt(0)),
			first.ElementAt(0).Text("a"),
			first.ElementAt(1).Text("b"),
			second.ElementAt(0).Text("c"),
		)
	})

	t.Run("exec environment is not CSV", func(t *testing.T) {
		t.Parallel()

		value := command.Flag("--exec-env").Value()
		candidate := value.CandidateAt(0)
		request := cli.InspectionRequest{
			Args: []string{"config", "set-credentials", "user", "--exec-env=TOKEN=a,b"},
		}
		clitest.Inspect(t.Context(), app, request).Require(t,
			clitest.InspectionReady(),
			value.Resolved(),
			value.CandidatesExactly(candidate),
			candidate.ElementCount(1),
			value.ElementsExactly(candidate.ElementAt(0)),
			candidate.ElementAt(0).Text("TOKEN=a,b"),
		)
	})
}

func TestConfigSetCredentialsPersistsResolvedCredentialCollections(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.RunRequest{
		Args: []string{
			"--kubeconfig", path,
			"config", "set-credentials", "user",
			"--auth-provider=oidc",
			"--auth-provider-arg=client-id=example",
			"--exec-command=plugin",
			"--exec-arg=arg1",
			"--exec-env=TOKEN=value",
		},
		Stdout: &bytes.Buffer{},
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	authInfo := config.AuthInfos["user"]
	if authInfo.AuthProvider == nil || authInfo.AuthProvider.Config["client-id"] != "example" {
		t.Fatalf("auth provider = %#v", authInfo.AuthProvider)
	}
	if authInfo.Exec == nil {
		t.Fatal("exec config is nil")
	}
	if len(authInfo.Exec.Args) != 1 || authInfo.Exec.Args[0] != "arg1" {
		t.Fatalf("exec args = %#v", authInfo.Exec.Args)
	}
	if len(authInfo.Exec.Env) != 1 || authInfo.Exec.Env[0].Name != "TOKEN" || authInfo.Exec.Env[0].Value != "value" {
		t.Fatalf("exec env = %#v", authInfo.Exec.Env)
	}
}

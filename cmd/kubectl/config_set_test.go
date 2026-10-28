package main

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestConfigSetDeclaresPropertyAndValueLanguage(t *testing.T) {
	t.Parallel()

	app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
		config.Subcommands.Command(configSetCommand(destinations.Kubeconfig))
	})
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "dotted map key",
			args:  []string{"config", "set", "clusters.dev.example.server", "https://127.0.0.1"},
			ready: true,
		},
		{
			name: "unsupported nested property",
			args: []string{"config", "set", "users.alice.exec.command", "plugin"},
		},
		{
			name: "unknown property root",
			args: []string{"config", "set", "user.bad", "value"},
		},
		{
			name: "empty property value",
			args: []string{"config", "set", "current-context", ""},
		},
		{
			name: "invalid bool property value",
			args: []string{"config", "set", "clusters.dev.insecure-skip-tls-verify", "sometimes"},
		},
		{
			name: "invalid base64 byte property value",
			args: []string{"config", "set", "clusters.dev.certificate-authority-data", "not base64!"},
		},
		{
			name:  "raw byte property value",
			args:  []string{"config", "set", "clusters.dev.certificate-authority-data", "not base64!", "--set-raw-bytes"},
			ready: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if test.ready {
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

func TestConfigSetMutatesKubeconfigProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		property string
		value    string
		flags    []string
		read     func(string) string
		want     string
	}{
		{
			name:     "current context",
			property: "current-context",
			value:    "other",
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				return config.CurrentContext
			},
			want: "other",
		},
		{
			name:     "dotted cluster name",
			property: "clusters.dev.example.server",
			value:    "https://example.invalid",
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				cluster := config.Clusters["dev.example"]
				if cluster == nil {
					return "cluster missing"
				}
				return cluster.Server
			},
			want: "https://example.invalid",
		},
		{
			name:     "boolean field",
			property: "clusters.cluster.insecure-skip-tls-verify",
			value:    "true",
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				if config.Clusters["cluster"].InsecureSkipTLSVerify {
					return "true"
				}
				return "false"
			},
			want: "true",
		},
		{
			name:     "base64 bytes",
			property: "clusters.cluster.certificate-authority-data",
			value:    base64.StdEncoding.EncodeToString([]byte("certificate")),
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				return string(config.Clusters["cluster"].CertificateAuthorityData)
			},
			want: "certificate",
		},
		{
			name:     "raw bytes",
			property: "clusters.cluster.certificate-authority-data",
			value:    "certificate",
			flags:    []string{"--set-raw-bytes"},
			read: func(path string) string {
				config, err := clientcmd.LoadFromFile(path)
				if err != nil {
					return "load error: " + err.Error()
				}
				return string(config.Clusters["cluster"].CertificateAuthorityData)
			},
			want: "certificate",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := writeConfigTestKubeconfig(t, "dev", "dev")
			app := compileConfigTestApplication(t, func(config *cli.CommandBuilder[struct{}], destinations configTestDestinations) {
				config.Subcommands.Command(configSetCommand(destinations.Kubeconfig))
			})
			args := []string{"--kubeconfig", path, "config", "set", test.property, test.value}
			args = append(args, test.flags...)
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

package main

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestConfigSetClusterDeclaresCertificatePolicies(t *testing.T) {
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
			name: "explicit false embed without certificate authority",
			args: []string{"config", "set-cluster", "cluster", "--embed-certs=false"},
		},
		{
			name:     "embed requires certificate authority",
			args:     []string{"config", "set-cluster", "cluster", "--embed-certs"},
			rejected: true,
		},
		{
			name:     "insecure conflicts with certificate authority",
			args:     []string{"config", "set-cluster", "cluster", "--insecure-skip-tls-verify", "--certificate-authority=ca.crt"},
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

func TestConfigSetClusterCertificateAuthorityUsesNativeFileCompletion(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.CompletionRequest{
		Args: []string{"config", "set-cluster", "cluster", "--certificate-authority", ""},
		Cursor: cli.CompletionCursor{
			Argument: 4,
		},
		Client: cli.CompletionClient{
			Capabilities: cli.CompletionClientCapabilities{
				Native: cli.CompletionNativeCapabilities{
					Files: true,
				},
			},
		},
	}
	clitest.Complete(t.Context(), app, request).Require(t,
		clitest.CompletionNativeRequestsExactly(clitest.CompletionNativeRequestAt(0)),
		clitest.CompletionNativeRequestAt(0).Kind(cli.CompletionNativeFiles),
	)
}

func TestConfigSetClusterPreservesBooleanFlagTriState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		starting bool
		arg      string
		want     bool
	}{
		{
			name:     "explicit false",
			starting: true,
			arg:      "--insecure-skip-tls-verify=false",
			want:     false,
		},
		{
			name: "bare flag",
			arg:  "--insecure-skip-tls-verify",
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := writeConfigTestKubeconfig(t, "dev", "dev")
			config, err := clientcmd.LoadFromFile(path)
			if err != nil {
				t.Fatalf("LoadFromFile() error = %v", err)
			}
			config.Clusters["cluster"].InsecureSkipTLSVerify = test.starting
			if err := clientcmd.WriteToFile(*config, path); err != nil {
				t.Fatalf("WriteToFile() error = %v", err)
			}

			app, err := application(pluginCatalog{})
			if err != nil {
				t.Fatalf("Application() error = %v", err)
			}
			request := cli.RunRequest{
				Args:   []string{"--kubeconfig", path, "config", "set-cluster", "cluster", test.arg},
				Stdout: &bytes.Buffer{},
			}
			if err := app.Run(t.Context(), request); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			config, err = clientcmd.LoadFromFile(path)
			if err != nil {
				t.Fatalf("LoadFromFile() error = %v", err)
			}
			if got := config.Clusters["cluster"].InsecureSkipTLSVerify; got != test.want {
				t.Fatalf("insecure-skip-tls-verify = %t, want %t", got, test.want)
			}
		})
	}
}

func TestConfigSetClusterPreservesPflagScalarRepeatBehavior(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{
			"config", "set-cluster", "cluster",
			"--insecure-skip-tls-verify=true",
			"--insecure-skip-tls-verify=false",
			"--certificate-authority=ca.crt",
		},
	}
	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

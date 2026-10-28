package kubectl_test

import (
	"bytes"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
	"k8s.io/client-go/tools/clientcmd"
)

func TestConfigUnsetDeclaresPropertyLanguage(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name  string
		path  string
		ready bool
	}{
		{
			name:  "whole exec property",
			path:  "users.alice.exec",
			ready: true,
		},
		{
			name:  "dotted map key",
			path:  "contexts.10.12.12.12",
			ready: true,
		},
		{
			name: "unsupported nested property",
			path: "users.alice.exec.command",
		},
		{
			name: "unknown property root",
			path: "user.bad",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: []string{"config", "unset", test.path},
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

func TestConfigUnsetMutatesKubeconfigProperties(t *testing.T) {
	t.Parallel()

	t.Run("current context", func(t *testing.T) {
		t.Parallel()

		path := writeConfigTestKubeconfig(t, "dev", "dev")
		app, err := kubectl.Application(kubectl.PluginCatalog{})
		if err != nil {
			t.Fatalf("Application() error = %v", err)
		}
		request := cli.RunRequest{
			Args:   []string{"--kubeconfig", path, "config", "unset", "current-context"},
			Stdout: &bytes.Buffer{},
		}
		if err := app.Run(t.Context(), request); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		config, err := clientcmd.LoadFromFile(path)
		if err != nil {
			t.Fatalf("LoadFromFile() error = %v", err)
		}
		if config.CurrentContext != "" {
			t.Fatalf("current context = %q, want empty", config.CurrentContext)
		}
	})

	t.Run("dotted context name", func(t *testing.T) {
		t.Parallel()

		path := writeConfigTestKubeconfig(t, "dev.example", "dev.example")
		app, err := kubectl.Application(kubectl.PluginCatalog{})
		if err != nil {
			t.Fatalf("Application() error = %v", err)
		}
		request := cli.RunRequest{
			Args:   []string{"--kubeconfig", path, "config", "unset", "contexts.dev.example"},
			Stdout: &bytes.Buffer{},
		}
		if err := app.Run(t.Context(), request); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		config, err := clientcmd.LoadFromFile(path)
		if err != nil {
			t.Fatalf("LoadFromFile() error = %v", err)
		}
		if _, ok := config.Contexts["dev.example"]; ok {
			t.Fatal("context dev.example still exists")
		}
	})

	t.Run("missing map key", func(t *testing.T) {
		t.Parallel()

		path := writeConfigTestKubeconfig(t, "dev", "dev")
		app, err := kubectl.Application(kubectl.PluginCatalog{})
		if err != nil {
			t.Fatalf("Application() error = %v", err)
		}
		request := cli.RunRequest{
			Args:   []string{"--kubeconfig", path, "config", "unset", "contexts.missing.namespace"},
			Stdout: &bytes.Buffer{},
		}
		if err := app.Run(t.Context(), request); err == nil {
			t.Fatal("Run() error = nil, want missing map key error")
		}
	})
}

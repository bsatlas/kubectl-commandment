package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestUseContextTreatsKnownNamesAsCompletionCandidates(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))

	inspectionRequest := cli.InspectionRequest{
		Args: []string{"--kubeconfig", path, "config", "use-context", "missing"},
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.InspectionReady())

	completionRequest := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "use-context", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItem("dev").Present())
}

func TestUseContextMutatesSelectedKubeconfig(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "old", "old", "dev")
	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))

	var stdout bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"--kubeconfig", path, "config", "use-context", "dev"},
		Stdout: &stdout,
		Stderr: io.Discard,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := stdout.String(), "Switched to context \"dev\".\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if got, want := config.CurrentContext, "dev"; got != want {
		t.Fatalf("current context = %q, want %q", got, want)
	}
}

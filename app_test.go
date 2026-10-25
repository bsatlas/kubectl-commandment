package kubectlcommandment_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	kubectlcommandment "github.com/abstractionsinc/kubectl-commandment"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestUseContextMutatesSelectedKubeconfig(t *testing.T) {
	path := writeKubeconfig(t, "old", "old", "dev")
	app, err := kubectlcommandment.Application()
	if err != nil {
		t.Fatalf("compile application: %v", err)
	}

	var stdout bytes.Buffer
	err = app.Run(context.Background(), runRequest(
		[]string{"config", "use-context", "dev", "--kubeconfig", path},
		&stdout,
	))
	if err != nil {
		t.Fatalf("run use-context: %v", err)
	}
	if got, want := stdout.String(), "Switched to context \"dev\".\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	loaded, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("load mutated kubeconfig: %v", err)
	}
	if got, want := loaded.CurrentContext, "dev"; got != want {
		t.Fatalf("current context = %q, want %q", got, want)
	}
}

func TestUseContextAliasUsesSameSemanticPath(t *testing.T) {
	path := writeKubeconfig(t, "old", "old", "dev")
	app, err := kubectlcommandment.Application()
	if err != nil {
		t.Fatalf("compile application: %v", err)
	}

	var stdout bytes.Buffer
	err = app.Run(context.Background(), runRequest(
		[]string{"--kubeconfig", path, "config", "use", "dev"},
		&stdout,
	))
	if err != nil {
		t.Fatalf("run use alias: %v", err)
	}
	if got, want := stdout.String(), "Switched to context \"dev\".\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestUseContextRejectsUnknownContextBeforeMutation(t *testing.T) {
	path := writeKubeconfig(t, "old", "old", "dev")
	app, err := kubectlcommandment.Application()
	if err != nil {
		t.Fatalf("compile application: %v", err)
	}

	var stdout bytes.Buffer
	err = app.Run(context.Background(), runRequest(
		[]string{"config", "use-context", "missing", "--kubeconfig", path},
		&stdout,
	))
	if err == nil {
		t.Fatal("run use-context unexpectedly succeeded")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	loaded, loadErr := clientcmd.LoadFromFile(path)
	if loadErr != nil {
		t.Fatalf("load kubeconfig: %v", loadErr)
	}
	if got, want := loaded.CurrentContext, "old"; got != want {
		t.Fatalf("current context = %q after rejected invocation, want %q", got, want)
	}
}

func writeKubeconfig(t *testing.T, current string, contexts ...string) string {
	t.Helper()
	config := clientcmdapi.NewConfig()
	config.CurrentContext = current
	for _, name := range contexts {
		config.Contexts[name] = &clientcmdapi.Context{Cluster: "cluster"}
	}
	config.Clusters["cluster"] = &clientcmdapi.Cluster{Server: "https://127.0.0.1"}

	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}

func TestUseContextCompletionUsesSelectedKubeconfig(t *testing.T) {
	path := writeKubeconfig(t, "old", "old", "dev", "staging")
	app, err := kubectlcommandment.Application()
	if err != nil {
		t.Fatalf("compile application: %v", err)
	}

	completion, err := app.Complete(context.Background(), cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "use-context", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	})
	if err != nil {
		t.Fatalf("complete context: %v", err)
	}
	if !completionContains(completion, "dev") {
		t.Fatalf("completion = %#v, want dev", completion.Items)
	}
	if completionContains(completion, "staging") {
		t.Fatalf("completion = %#v, staging should not match prefix d", completion.Items)
	}
}

func TestCurrentContextReadsSelectedKubeconfig(t *testing.T) {
	path := writeKubeconfig(t, "dev", "old", "dev")
	app, err := kubectlcommandment.Application()
	if err != nil {
		t.Fatalf("compile application: %v", err)
	}

	var stdout bytes.Buffer
	err = app.Run(context.Background(), runRequest(
		[]string{"config", "current-context", "--kubeconfig", path},
		&stdout,
	))
	if err != nil {
		t.Fatalf("run current-context: %v", err)
	}
	if got, want := stdout.String(), "dev\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestCurrentContextRejectsUnsetContext(t *testing.T) {
	path := writeKubeconfig(t, "", "dev")
	app, err := kubectlcommandment.Application()
	if err != nil {
		t.Fatalf("compile application: %v", err)
	}

	var stdout bytes.Buffer
	err = app.Run(context.Background(), runRequest(
		[]string{"--kubeconfig", path, "config", "current-context"},
		&stdout,
	))
	if err == nil {
		t.Fatal("run current-context unexpectedly succeeded")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func completionContains(completion cli.Completion, value string) bool {
	for _, item := range completion.Items {
		if item.Value == value {
			return true
		}
	}
	return false
}

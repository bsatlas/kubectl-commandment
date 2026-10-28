package main

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestUseContextMutatesSelectedKubeconfig(t *testing.T) {
	t.Parallel()

	path := writeTestKubeconfig(t, "old", "old", "dev")
	app := compileUseContextTestApplication(t)

	var stdout bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"config", "use-context", "dev", "--kubeconfig", path},
		Stdout: &stdout,
		Stderr: io.Discard,
	}
	err := app.Run(t.Context(), request)
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

func TestUseContextCompletionUsesSelectedKubeconfig(t *testing.T) {
	t.Parallel()

	path := writeTestKubeconfig(t, "old", "old", "dev", "staging")
	app := compileUseContextTestApplication(t)

	request := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "use-context", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	}
	completion, err := app.Complete(t.Context(), request)
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

func compileUseContextTestApplication(t *testing.T) cli.Application {
	t.Helper()
	definition := cli.Definition[rootConfig]{
		Name: "kubectl",
		Declare: func(root *cli.CommandBuilder[rootConfig]) {
			root.Dialect.FlagOrder(cli.FlagOrderInterspersed)
			kubeconfig := root.Values.String(&root.Config.Kubeconfig)
			root.Flags.ValueFor(kubeconfig).Placement(cli.FlagPlacementSubtree).Long("kubeconfig")
			configCommand := cli.Command[struct{}]{
				Name: "config",
				Declare: func(config *cli.CommandBuilder[struct{}]) {
					config.Subcommands.Command(useContextCommand(kubeconfig))
				},
			}
			root.Subcommands.Command(configCommand)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("compile test application: %v", err)
	}
	return app
}

func writeTestKubeconfig(t *testing.T, current string, contexts ...string) string {
	t.Helper()
	config := api.NewConfig()
	config.CurrentContext = current
	for _, name := range contexts {
		config.Contexts[name] = &api.Context{
			Cluster: "cluster",
		}
	}
	config.Clusters["cluster"] = &api.Cluster{
		Server: "https://127.0.0.1",
	}

	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}

func completionContains(completion cli.Completion, value string) bool {
	for _, item := range completion.Items {
		if item.Value == value {
			return true
		}
	}
	return false
}

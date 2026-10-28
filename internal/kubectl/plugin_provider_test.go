package kubectl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestPluginProvidersIgnoreStaticCollisionsAndBuildNestedCommands(t *testing.T) {
	t.Parallel()

	pluginDir := t.TempDir()
	writePlugin(t, pluginDir, "kubectl-get")
	writePlugin(t, pluginDir, "kubectl-foo-bar")
	plugins := kubectl.DiscoverPluginCatalog(pluginDir)
	app, err := kubectl.Application(plugins)
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	schema := app.Schema()
	var get cli.CommandSchema
	for _, id := range schema.CommandOrder {
		command := schema.Commands[id]
		if len(command.Path) == 1 && command.Name == "get" {
			get = command
			break
		}
	}
	if get.ID == 0 {
		t.Fatal("get command is missing")
	}
	if got, want := get.Description.Short, "Display one or many resources"; got != want {
		t.Fatalf("get description = %q, want %q", got, want)
	}

	request := cli.InspectionRequest{
		Args: []string{"foo", "bar", "arg"},
	}
	inspection, err := app.Inspect(t.Context(), request)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if got, want := len(inspection.Selection.Command.Path), 3; got != want {
		t.Fatalf("selection path length = %d, want %d", got, want)
	}
}

func TestPluginProviderDelegatesArgumentCompletionToPluginProtocol(t *testing.T) {
	t.Parallel()

	pluginDir := t.TempDir()
	writePlugin(t, pluginDir, "kubectl-foo_bar-baz")
	plugins := kubectl.DiscoverPluginCatalog(pluginDir)
	app, err := kubectl.Application(plugins)
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	completion, err := app.Complete(t.Context(), cli.CompletionRequest{
		Args: []string{"foo-bar", "baz", "left", "ri"},
		Cursor: cli.CompletionCursor{
			Argument: 3,
			Byte:     2,
		},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if completion.Delegation == nil {
		t.Fatal("completion delegation = nil")
	}
	delegation := completion.Delegation
	if got, want := len(delegation.Args), 3; got != want {
		t.Fatalf("delegation args length = %d, want %d: %#v", got, want, delegation.Args)
	}
	if got, want := delegation.Args[0], "kubectl_complete-foo_bar-baz"; got != want {
		t.Fatalf("delegation executable = %q, want %q", got, want)
	}
	if got, want := delegation.Args[1], "left"; got != want {
		t.Fatalf("delegation arg[1] = %q, want %q", got, want)
	}
	if got, want := delegation.Args[2], "ri"; got != want {
		t.Fatalf("delegation arg[2] = %q, want %q", got, want)
	}
	wantCursor := cli.CompletionCursor{
		Argument: 2,
		Byte:     2,
	}
	if got, want := delegation.Cursor, wantCursor; got != want {
		t.Fatalf("delegation cursor = %#v, want %#v", got, want)
	}
}

func writePlugin(t *testing.T, dir, name string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", name, err)
	}
}

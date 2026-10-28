package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestPluginProvidersIgnoreStaticCollisionsAndBuildNestedCommands(t *testing.T) {
	t.Parallel()

	definition := cli.Definition[struct{}]{
		Name: "kubectl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			get := cli.Command[struct{}]{
				Name: "get",
			}
			root.Subcommands.Command(get)
			nodes := []pluginNode{
				{
					Name:       "get",
					LookupName: "get",
				},
				{
					Name:       "foo",
					LookupName: "foo",
					Children: []pluginNode{
						{
							Name:       "bar",
							LookupName: "foo-bar",
						},
					},
				},
			}
			addPluginProviders(&root.Subcommands, nodes, pluginOccupiedNames(get.Name))
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
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

	definition := cli.Definition[struct{}]{
		Name: "kubectl",
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			addPluginProviders(&root.Subcommands, []pluginNode{
				{
					Name: "foo",
					Children: []pluginNode{
						{
							Name:       "bar",
							LookupName: "foo_bar-baz",
						},
					},
				},
			}, nil)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatal(err)
	}

	completion, err := app.Complete(t.Context(), cli.CompletionRequest{
		Args: []string{"foo", "bar", "left", "ri"},
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

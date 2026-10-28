package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestOptionsUsesStructuralHelpForInheritedFlags(t *testing.T) {
	t.Parallel()

	type config struct {
		Kubeconfig string
	}
	definition := cli.Definition[config]{
		Name: "kubectl",
		Help: cli.HelpConfig{
			Renderer: renderHelp,
		},
		Declare: func(root *cli.CommandBuilder[config]) {
			kubeconfig := root.Flags.String(&root.Config.Kubeconfig)
			kubeconfig.Placement(cli.FlagPlacementSubtree).Long("kubeconfig")
			root.Subcommands.Command(optionsCommand)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var output bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"options"},
		Stdout: &output,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run(options) error = %v", err)
	}
	text := output.String()
	if !strings.Contains(text, "The following options can be passed to any command") {
		t.Fatalf("Run(options) output missing inherited-options heading:\n%s", text)
	}
	if !strings.Contains(text, "--kubeconfig") {
		t.Fatalf("Run(options) output missing inherited --kubeconfig flag:\n%s", text)
	}
	if strings.Contains(text, "--help") {
		t.Fatalf("Run(options) output unexpectedly includes Help action flag:\n%s", text)
	}
}

func TestAlphaUsesStructuralHelpForEmptyAlphaNamespace(t *testing.T) {
	t.Parallel()

	definition := cli.Definition[struct{}]{
		Name: "kubectl",
		Help: cli.HelpConfig{
			Renderer: renderHelp,
		},
		Declare: func(root *cli.CommandBuilder[struct{}]) {
			root.Subcommands.Command(alphaCommand)
		},
	}
	app, err := cli.Compile(definition)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var output bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"alpha"},
		Stdout: &output,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run(alpha) error = %v", err)
	}
	if got, want := output.String(), "No alpha commands are available in this version of kubectl\n"; got != want {
		t.Fatalf("Run(alpha) output = %q, want %q", got, want)
	}
}

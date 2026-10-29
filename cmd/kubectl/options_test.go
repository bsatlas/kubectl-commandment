package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestOptionsUsesStructuralHelpForInheritedFlags(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
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

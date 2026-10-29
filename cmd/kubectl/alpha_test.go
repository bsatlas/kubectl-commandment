package main

import (
	"bytes"

	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestAlphaUsesStructuralHelpForEmptyAlphaNamespace(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
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

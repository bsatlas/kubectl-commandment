package main

import (
	"context"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestRunSeparatorPreservesRequiredNameAndTransfersPostBoundaryArguments(t *testing.T) {
	t.Parallel()

	var got runConfig
	app := compileSeparatorTestApplication(t, runCommand, func(_ context.Context, cfg *runConfig) error {
		got = *cfg
		return nil
	})

	err := app.Run(t.Context(), cli.RunRequest{Args: []string{"run", "work", "--", "arg", "--image=busybox"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Name != "work" {
		t.Fatalf("name = %q, want work", got.Name)
	}
	if want := []string{"arg", "--image=busybox"}; !slices.Equal(got.Arguments, want) {
		t.Fatalf("arguments = %#v, want %#v", got.Arguments, want)
	}
	if got.Image != "" {
		t.Fatalf("image = %q, want empty because post-boundary flag text is transferred", got.Image)
	}

	err = app.Run(t.Context(), cli.RunRequest{Args: []string{"run", "--", "work"}})
	if err == nil {
		t.Fatal("Run() with NAME only after separator succeeded, want required pre-boundary NAME rejection")
	}
}

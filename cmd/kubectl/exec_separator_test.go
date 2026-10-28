package main

import (
	"context"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestExecFormLocalSeparatorTransfersKnownFlagText(t *testing.T) {
	t.Parallel()

	var got execConfig
	app := compileSeparatorTestApplication(t, execCommand, func(_ context.Context, cfg *execConfig) error {
		got = *cfg
		return nil
	})

	err := app.Run(t.Context(), cli.RunRequest{Args: []string{"exec", "pod-a", "--", "sh", "--container=remote"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Resource != "pod-a" {
		t.Fatalf("resource = %q, want pod-a", got.Resource)
	}
	if want := []string{"sh", "--container=remote"}; !slices.Equal(got.Command, want) {
		t.Fatalf("command = %#v, want %#v", got.Command, want)
	}
	if got.Container != "" {
		t.Fatalf("container = %q, want empty because post-boundary flag text is transferred", got.Container)
	}

	err = app.Run(t.Context(), cli.RunRequest{Args: []string{"exec", "pod-a", "sh"}})
	if err == nil {
		t.Fatal("Run() without required exec separator succeeded")
	}
}

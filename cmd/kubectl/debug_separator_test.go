package main

import (
	"context"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestDebugSeparatorTransfersOnlyPostBoundaryArguments(t *testing.T) {
	t.Parallel()

	var got debugConfig
	app := compileSeparatorTestApplication(t, debugCommand, func(_ context.Context, cfg *debugConfig) error {
		got = *cfg
		return nil
	})

	err := app.Run(t.Context(), cli.RunRequest{Args: []string{"debug", "pod-a", "pod-b", "--", "sh", "--image=busybox"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if want := []string{"pod-a", "pod-b"}; !slices.Equal(got.TargetNames, want) {
		t.Fatalf("targets = %#v, want %#v", got.TargetNames, want)
	}
	if want := []string{"sh", "--image=busybox"}; !slices.Equal(got.Arguments, want) {
		t.Fatalf("arguments = %#v, want %#v", got.Arguments, want)
	}
	if got.Image != "" {
		t.Fatalf("image = %q, want empty because post-boundary flag text is transferred", got.Image)
	}

	got = debugConfig{}
	err = app.Run(t.Context(), cli.RunRequest{Args: []string{"debug", "pod-a", "pod-b"}})
	if err != nil {
		t.Fatalf("Run() without separator error = %v", err)
	}
	if want := []string{"pod-a", "pod-b"}; !slices.Equal(got.TargetNames, want) {
		t.Fatalf("targets without separator = %#v, want %#v", got.TargetNames, want)
	}
	if len(got.Arguments) != 0 {
		t.Fatalf("arguments without separator = %#v, want none", got.Arguments)
	}
}

package main

import (
	"context"
	"slices"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
)

func TestCreateDeploymentRequiresSeparatorBeforeCommand(t *testing.T) {
	t.Parallel()

	var got createDeploymentConfig
	app := compileSeparatorTestApplication(t, createDeploymentCommand, func(_ context.Context, cfg *createDeploymentConfig) error {
		got = *cfg
		return nil
	})

	err := app.Run(t.Context(), cli.RunRequest{Args: []string{"deployment", "work", "--", "date", "--image=remote"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Name != "work" {
		t.Fatalf("name = %q, want work", got.Name)
	}
	if want := []string{"date", "--image=remote"}; !slices.Equal(got.Command, want) {
		t.Fatalf("command = %#v, want %#v", got.Command, want)
	}
	if len(got.Images) != 0 {
		t.Fatalf("images = %#v, want none because post-boundary flag text is transferred", got.Images)
	}

	err = app.Run(t.Context(), cli.RunRequest{Args: []string{"deployment", "work", "date"}})
	if err == nil {
		t.Fatal("Run() accepted deployment command without separator")
	}
}

func TestCreateNameOnlyCommandAcceptsButDoesNotInterpretPostSeparatorArguments(t *testing.T) {
	t.Parallel()

	var got createNamespaceConfig
	app := compileSeparatorTestApplication(t, createNamespaceCommand, func(_ context.Context, cfg *createNamespaceConfig) error {
		got = *cfg
		return nil
	})

	err := app.Run(t.Context(), cli.RunRequest{Args: []string{"namespace", "work", "--", "tail", "--dry-run=client"}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Name != "work" {
		t.Fatalf("name = %q, want work", got.Name)
	}
	if want := []string{"tail", "--dry-run=client"}; !slices.Equal(got.TrailingArguments, want) {
		t.Fatalf("trailing arguments = %#v, want %#v", got.TrailingArguments, want)
	}
	if got.DryRun != "none" {
		t.Fatalf("dry-run = %q, want default none because post-boundary flag text is transferred", got.DryRun)
	}

	err = app.Run(t.Context(), cli.RunRequest{Args: []string{"namespace", "work", "tail"}})
	if err == nil {
		t.Fatal("Run() accepted extra pre-separator namespace argument")
	}
}

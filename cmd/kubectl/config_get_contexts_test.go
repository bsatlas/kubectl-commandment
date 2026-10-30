package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestConfigGetContextsOwnsItsOutputLanguage(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	readyRequest := cli.InspectionRequest{
		Args: []string{"config", "get-contexts", "missing", "also-missing", "--output=name"},
	}
	clitest.Inspect(t.Context(), app, readyRequest).Require(t, clitest.InspectionReady())

	rejectedRequest := cli.InspectionRequest{
		Args: []string{"config", "get-contexts", "--output=json"},
	}
	clitest.Inspect(t.Context(), app, rejectedRequest).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
	)
}

func TestConfigGetContextsDoesNotCompleteContextNames(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "get-contexts", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	}
	clitest.Complete(t.Context(), app, request).Require(t, clitest.CompletionItem("dev").Absent())
}

func TestConfigGetContextsPreservesPflagScalarBehavior(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	for _, args := range [][]string{
		{"config", "get-contexts", "-oname"},
	} {
		request := cli.InspectionRequest{Args: args}
		clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
	}

	var stdout bytes.Buffer
	request := cli.RunRequest{
		Args: []string{
			"--kubeconfig", path,
			"config", "get-contexts",
			"--no-headers=true",
			"--no-headers=false",
		},
		Stdout: &stdout,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.HasPrefix(stdout.String(), "CURRENT") {
		t.Fatalf("stdout = %q, want header from final --no-headers=false", stdout.String())
	}
}

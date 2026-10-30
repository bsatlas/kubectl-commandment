package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRolloutUndoDryRunDomain(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	cases := []struct {
		name string
		args []string
		want clitest.InspectionExpectation
	}{
		{
			name: "none",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=none"},
			want: clitest.InspectionReady(),
		},
		{
			name: "client",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=client"},
			want: clitest.InspectionReady(),
		},
		{
			name: "server",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=server"},
			want: clitest.InspectionReady(),
		},
		{
			name: "legacy true",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=true"},
			want: clitest.InspectionReady(),
		},
		{
			name: "legacy false",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=false"},
			want: clitest.InspectionReady(),
		},
		{
			name: "unknown",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=remote"},
			want: clitest.InspectionRejected(),
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			proof := clitest.Inspect(t.Context(), app, request)
			proof.Require(t, test.want)
		})
	}
}

func TestRolloutUndoDeclaresUpstreamResourceTypeCompletion(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.CompletionRequest{
		Args:   []string{"rollout", "undo", ""},
		Cursor: cli.CompletionCursor{Argument: 2, Byte: 0},
	}
	clitest.Complete(t.Context(), app, request).Require(t,
		clitest.CompletionItem("deployment").Present(),
		clitest.CompletionItem("daemonset").Present(),
		clitest.CompletionItem("statefulset").Present(),
	)
}

func TestRolloutUndoDryRunImplicitValueIsClient(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	dryRun := clitest.Command("rollout", "undo").Flag("--dry-run").Value()
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "undo", "deployment/frontend", "--dry-run"},
	}).Require(t,
		clitest.InspectionReady(),
		dryRun.Text("client"),
	)
}

func TestRolloutUndoRejectsNegativeRevision(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "undo", "deployment/frontend", "--to-revision", "-1"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestRolloutUndoUsesLastRepeatedScalarValue(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	command := clitest.Command("rollout", "undo")
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{
			"rollout", "undo", "deployment/frontend",
			"--to-revision=1", "--to-revision=2",
			"--selector=app=old", "--selector=app=new",
			"--dry-run=client", "--dry-run=server",
		},
	}).Require(t,
		clitest.InspectionReady(),
		command.Flag("--to-revision").Value().Text("2"),
		command.Flag("--selector").Value().Text("app=new"),
		command.Flag("--dry-run").Value().Text("server"),
	)
}

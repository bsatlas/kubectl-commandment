package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRolloutHistoryRejectsNegativeRevision(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "history", "deployment/frontend", "--revision", "-1"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestRolloutHistoryDeclaresUpstreamResourceTypeCompletion(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.CompletionRequest{
		Args:   []string{"rollout", "history", ""},
		Cursor: cli.CompletionCursor{Argument: 2, Byte: 0},
	}
	clitest.Complete(t.Context(), app, request).Require(t,
		clitest.CompletionItem("deployment").Present(),
		clitest.CompletionItem("daemonset").Present(),
		clitest.CompletionItem("statefulset").Present(),
	)
}

func TestRolloutHistoryUsesLastRepeatedScalarValue(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{
			"rollout", "history", "deployment/frontend",
			"--revision=1", "--revision=2",
			"--selector=app=old", "--selector=app=new",
		},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.Command("rollout", "history").Flag("--revision").Value().Text("2"),
		clitest.Command("rollout", "history").Flag("--selector").Value().Text("app=new"),
	)
}

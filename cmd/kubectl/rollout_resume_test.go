package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRolloutResumeDeclaresDeploymentCompletionAndFieldManagerDefault(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"rollout", "resume", "deployment/frontend"}}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.Command("rollout", "resume").Flag("--field-manager").Value().Text("kubectl-rollout"),
	)

	completion := cli.CompletionRequest{
		Args:   []string{"rollout", "resume", "d"},
		Cursor: cli.CompletionCursor{Argument: 2, Byte: 1},
	}
	clitest.Complete(t.Context(), app, completion).Require(t,
		clitest.CompletionItem("deployment").Present(),
		clitest.CompletionItem("daemonset").Absent(),
	)
}

func TestRolloutResumeUsesLastRepeatedScalarValue(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{
			"rollout", "resume", "deployment/frontend",
			"--field-manager=old", "--field-manager=new",
			"--selector=app=old", "--selector=app=new",
		},
	}).Require(t,
		clitest.InspectionReady(),
		clitest.Command("rollout", "resume").Flag("--field-manager").Value().Text("new"),
		clitest.Command("rollout", "resume").Flag("--selector").Value().Text("app=new"),
	)
}

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRolloutStatusAcceptsRepeatedResources(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "status", "deployment/frontend", "statefulset/backend"},
	})
	proof.Require(t, clitest.InspectionReady())
}

func TestRolloutStatusRejectsNegativeRevision(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "status", "deployment/frontend", "--revision", "-1"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

func TestRolloutStatusDeclaresWatchAndTimeoutValueForms(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	command := clitest.Command("rollout", "status")
	watch := command.Flag("--watch").Value()
	timeout := command.Flag("--timeout").Value()
	tests := []struct {
		name         string
		args         []string
		expectations []clitest.InspectionExpectation
	}{
		{
			name: "default watch",
			args: []string{"rollout", "status", "deployment/frontend"},
			expectations: []clitest.InspectionExpectation{
				clitest.InspectionReady(),
				watch.Text("true"),
			},
		},
		{
			name: "long assigned false",
			args: []string{"rollout", "status", "deployment/frontend", "--watch=false"},
			expectations: []clitest.InspectionExpectation{
				clitest.InspectionReady(),
				watch.Text("false"),
			},
		},
		{
			name: "short assigned false",
			args: []string{"rollout", "status", "deployment/frontend", "-w=false"},
			expectations: []clitest.InspectionExpectation{
				clitest.InspectionReady(),
				watch.Text("false"),
			},
		},
		{
			name: "typed timeout",
			args: []string{"rollout", "status", "deployment/frontend", "--timeout=2m"},
			expectations: []clitest.InspectionExpectation{
				clitest.InspectionReady(),
				timeout.Text("2m0s"),
			},
		},
		{
			name: "repeated scalar values keep last",
			args: []string{
				"rollout", "status", "deployment/frontend",
				"--revision=1", "--revision=2",
				"--selector=app=old", "--selector=app=new",
				"--timeout=1m", "--timeout=2m",
				"--watch=false", "--watch=true",
			},
			expectations: []clitest.InspectionExpectation{
				clitest.InspectionReady(),
				command.Flag("--revision").Value().Text("2"),
				command.Flag("--selector").Value().Text("app=new"),
				timeout.Text("2m0s"),
				watch.Text("true"),
			},
		},
		{
			name: "invalid timeout",
			args: []string{"rollout", "status", "deployment/frontend", "--timeout=forever"},
			expectations: []clitest.InspectionExpectation{
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticInvalidValue).Present()),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{Args: test.args}
			clitest.Inspect(t.Context(), app, request).Require(t, test.expectations...)
		})
	}
}

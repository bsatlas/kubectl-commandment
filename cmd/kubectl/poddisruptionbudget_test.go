package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestPoddisruptionbudgetFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "pdb requires selector",
			args: []string{"create", "poddisruptionbudget", "web", "--min-available=1"},
		},
		{
			name: "pdb availability is exclusive",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app=web", "--min-available=1", "--max-unavailable=1"},
		},
		{
			name: "pdb availability has kubectl syntax",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app=web", "--min-available=1Gi"},
		},
		{
			name: "pdb requires availability",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app=web"},
		},
		{
			name: "pdb selector has kubectl syntax",
			args: []string{"create", "poddisruptionbudget", "web", "--selector=app in (", "--min-available=1"},
		},
		{
			name: "pdb validates nonempty name",
			args: []string{"create", "poddisruptionbudget", "", "--selector=app=web", "--min-available=1"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{Args: test.args}
			clitest.Inspect(t.Context(), app, request).Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

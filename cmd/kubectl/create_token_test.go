package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateTokenFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "token validates nonempty name",
			args: []string{"create", "token", ""},
		},
		{
			name: "token duration is nonnegative",
			args: []string{"create", "token", "builder", "--duration=-1s"},
		},
		{
			name: "token duration is whole seconds",
			args: []string{"create", "token", "builder", "--duration=1500ms"},
		},
		{
			name: "token bound name requires kind",
			args: []string{"create", "token", "builder", "--bound-object-name=pod"},
		},
		{
			name: "token bound kind requires name",
			args: []string{"create", "token", "builder", "--bound-object-kind=Pod"},
		},
		{
			name: "token bound kind is constrained",
			args: []string{"create", "token", "builder", "--bound-object-kind=Deployment", "--bound-object-name=web"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestCreateTokenFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name         string
		args         []string
		expectations []clitest.InspectionExpectation
	}{
		{
			name: "token bound UID is optional",
			args: []string{"create", "token", "builder", "--bound-object-kind=Pod", "--bound-object-name=web"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			expectations := []clitest.InspectionExpectation{clitest.InspectionReady()}
			expectations = append(expectations, test.expectations...)
			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(t, expectations...)
		})
	}
}

func TestCreateTokenFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "token", "builder", "--bound-object-kind=Pod", "--bound-object-name=first", "--bound-object-name=second"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "token").Flag("--bound-object-name").Value().Text("second"),
	)
}

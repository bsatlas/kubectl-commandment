package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreatePriorityClassFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
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
			name: "priorityclass value remains optional",
			args: []string{"create", "priorityclass", "normal"},
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

func TestCreatePriorityClassFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "priorityclass", "normal", "--description=first", "--description=second"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "priorityclass").Flag("--description").Value().Text("second"),
	)
}

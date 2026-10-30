package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateNamespaceFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
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
			name: "subcommand does not inherit direct create source requirement",
			args: []string{"create", "namespace", "work"},
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

func TestCreateNamespaceFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "namespace", "team", "--field-manager=first", "--field-manager=second"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "namespace").Flag("--field-manager").Value().Text("second"),
	)
}

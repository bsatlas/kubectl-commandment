package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestClusteripFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "clusterip requires tcp unless headless",
			args: []string{"create", "service", "clusterip", "api"},
		},
		{
			name: "clusterip port has kubectl syntax",
			args: []string{"create", "service", "clusterip", "api", "--tcp=not-a-port"},
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

func TestClusteripFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
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
			name: "headless clusterip does not require tcp",
			args: []string{"create", "service", "clusterip", "api", "--clusterip=None"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			expectations := []clitest.InspectionExpectation{clitest.InspectionReady()}
			expectations = append(expectations, test.expectations...)
			request := cli.InspectionRequest{Args: test.args}
			clitest.Inspect(t.Context(), app, request).Require(t, expectations...)
		})
	}
}

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "create requires a source",
			args: []string{"create"},
		},
		{
			name: "create source is exclusive",
			args: []string{"create", "-f", "object.yaml", "-k", "overlay"},
		},
		{
			name: "kustomize rejects recursive",
			args: []string{"create", "-k", "overlay", "-R"},
		},
		{
			name: "raw rejects edit",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "--edit"},
		},
		{
			name: "raw requires one local file",
			args: []string{"create", "-f", "one.yaml", "-f", "two.yaml", "--raw=/api/v1/namespaces"},
		},
		{
			name: "raw rejects remote file",
			args: []string{"create", "--filename=https://example.invalid/object.yaml", "--raw=/api/v1/namespaces"},
		},
		{
			name: "raw rejects recursive",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-R"},
		},
		{
			name: "raw rejects selector",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-l", "app=web"},
		},
		{
			name: "raw rejects output",
			args: []string{"create", "-f", "object.yaml", "--raw=/api/v1/namespaces", "-o", "yaml"},
		},
		{
			name: "raw URI has kubectl syntax",
			args: []string{"create", "-f", "object.yaml", "--raw=%"},
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

func TestCreateFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
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
			name: "direct create accepts filename source",
			args: []string{"create", "-f", "object.yaml"},
		},
		{
			name: "direct create accepts assigned shorthand filename",
			args: []string{"create", "-f=object.yaml"},
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

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestIngressFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "ingress requires routing",
			args: []string{"create", "ingress", "web"},
		},
		{
			name: "ingress rejects malformed rule",
			args: []string{"create", "ingress", "web", "--rule=not-a-rule"},
		},
		{
			name: "ingress annotation has kubectl syntax",
			args: []string{"create", "ingress", "web", "--rule=example.com/=api:80", "--annotation=key"},
		},
		{
			name: "ingress backend has kubectl syntax",
			args: []string{"create", "ingress", "web", "--default-backend=api"},
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

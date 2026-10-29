package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRoleFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "role requires verbs",
			args: []string{"create", "role", "reader", "--resource=pods"},
		},
		{
			name: "role requires resources",
			args: []string{"create", "role", "reader", "--verb=get"},
		},
		{
			name: "role resource has kubectl syntax",
			args: []string{"create", "role", "reader", "--verb=get", "--resource=.apps"},
		},
		{
			name: "role validates nonempty name",
			args: []string{"create", "role", "", "--verb=get", "--resource=pods"},
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

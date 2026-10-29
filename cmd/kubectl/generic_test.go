package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestGenericFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "secret env file excludes literal",
			args: []string{"create", "secret", "generic", "settings", "--from-env-file=.env", "--from-literal=a=b"},
		},
		{
			name: "secret generic validates nonempty name",
			args: []string{"create", "secret", "generic", "", "--from-literal=a=b"},
		},
		{
			name: "secret generic file source has kubectl syntax",
			args: []string{"create", "secret", "generic", "settings", "--from-file==path"},
		},
		{
			name: "secret generic literal source has kubectl syntax",
			args: []string{"create", "secret", "generic", "settings", "--from-literal==value"},
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

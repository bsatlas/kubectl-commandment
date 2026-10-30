package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateConfigMapFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "configmap env file excludes file",
			args: []string{"create", "configmap", "settings", "--from-env-file=.env", "--from-file=settings.ini"},
		},
		{
			name: "configmap validates nonempty name",
			args: []string{"create", "configmap", "", "--from-literal=a=b"},
		},
		{
			name: "configmap file source has kubectl syntax",
			args: []string{"create", "configmap", "settings", "--from-file==path"},
		},
		{
			name: "configmap literal source has kubectl syntax",
			args: []string{"create", "configmap", "settings", "--from-literal==value"},
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

func TestCreateConfigMapFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "configmap", "settings", "--field-manager=first", "--field-manager=second"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "configmap").Flag("--field-manager").Value().Text("second"),
	)
}

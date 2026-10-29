package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestWaitDeclaresSourceConditionAndSelectionGrammar(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "resource condition",
			args:  []string{"wait", "pod/foo", "--for=create"},
			ready: true,
		},
		{
			name:  "filename condition",
			args:  []string{"wait", "-f", "pod.yaml", "--for=delete"},
			ready: true,
		},
		{
			name:  "negative timeout",
			args:  []string{"wait", "pod/foo", "--for=create", "--timeout=-1s"},
			ready: true,
		},
		{
			name:  "condition is optional upstream",
			args:  []string{"wait", "pod/foo"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"wait", "--for=create"},
			ready: false,
		},
		{
			name:  "all with selector follows upstream builder semantics",
			args:  []string{"wait", "pods", "--all", "--selector=app=web", "--for=create"},
			ready: true,
		},
		{
			name:  "all with field selector follows upstream builder semantics",
			args:  []string{"wait", "pods", "--all", "--field-selector=status.phase=Running", "--for=create"},
			ready: true,
		},
		{
			name:  "local resource argument",
			args:  []string{"wait", "pod/foo", "--local", "--for=create"},
			ready: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if test.ready {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

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
			name:  "invalid output format",
			args:  []string{"wait", "pod/foo", "--for=create", "--output=wat"},
			ready: false,
		},
		{
			name:  "resource condition",
			args:  []string{"wait", "pod/foo", "--for=create"},
			ready: true,
		},
		{
			name:  "assigned short filename",
			args:  []string{"wait", "-f=pod.yaml", "--for=create"},
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
			name:  "missing condition",
			args:  []string{"wait", "pod/foo"},
			ready: false,
		},
		{
			name:  "unknown condition",
			args:  []string{"wait", "pod/foo", "--for=wat"},
			ready: false,
		},
		{
			name:  "invalid jsonpath condition",
			args:  []string{"wait", "pod/foo", "--for=jsonpath="},
			ready: false,
		},
		{
			name:  "status condition",
			args:  []string{"wait", "pod/foo", "--for=condition=Ready=false"},
			ready: true,
		},
		{
			name:  "jsonpath condition",
			args:  []string{"wait", "pod/foo", "--for=jsonpath={.status.phase}=Running"},
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

func TestWaitRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"wait", "pod/foo", "--for=create", "--timeout=1s", "--timeout=2s"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("wait").Flag("--timeout").Value().Text("2s"),
	)
}

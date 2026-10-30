package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCordonDeclaresNodeSelectionGrammar(t *testing.T) {
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
			name:  "invalid dry run",
			args:  []string{"cordon", "node-a", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "node argument",
			args:  []string{"cordon", "node-a"},
			ready: true,
		},
		{
			name:  "assigned short selector",
			args:  []string{"cordon", "-l=role=worker"},
			ready: true,
		},
		{
			name:  "selector",
			args:  []string{"cordon", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "missing selection",
			args:  []string{"cordon"},
			ready: false,
		},
		{
			name:  "argument and selector",
			args:  []string{"cordon", "node-a", "--selector=role=worker"},
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

func TestCordonRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"cordon", "--selector=role=one", "--selector=role=two"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("cordon").Flag("--selector").Value().Text("role=two"),
	)
}

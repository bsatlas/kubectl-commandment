package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestUncordonDeclaresNodeSelectionGrammar(t *testing.T) {
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
			args:  []string{"uncordon", "node-a", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "node argument",
			args:  []string{"uncordon", "node-a"},
			ready: true,
		},
		{
			name:  "assigned short selector",
			args:  []string{"uncordon", "-l=role=worker"},
			ready: true,
		},
		{
			name:  "selector",
			args:  []string{"uncordon", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "missing selection",
			args:  []string{"uncordon"},
			ready: false,
		},
		{
			name:  "argument and selector",
			args:  []string{"uncordon", "node-a", "--selector=role=worker"},
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

func TestUncordonRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"uncordon", "--selector=role=one", "--selector=role=two"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("uncordon").Flag("--selector").Value().Text("role=two"),
	)
}

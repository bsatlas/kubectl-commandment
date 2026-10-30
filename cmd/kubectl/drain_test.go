package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestDrainDeclaresNodeSelectionAndPodSelectorLanguage(t *testing.T) {
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
			args:  []string{"drain", "node-a", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "node argument",
			args:  []string{"drain", "node-a"},
			ready: true,
		},
		{
			name:  "assigned short selector",
			args:  []string{"drain", "-l=role=worker"},
			ready: true,
		},
		{
			name:  "node selector",
			args:  []string{"drain", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "missing selection",
			args:  []string{"drain"},
			ready: false,
		},
		{
			name:  "argument and selector",
			args:  []string{"drain", "node-a", "--selector=role=worker"},
			ready: false,
		},
		{
			name:  "valid pod selector",
			args:  []string{"drain", "node-a", "--pod-selector=app=web"},
			ready: true,
		},
		{
			name:  "invalid pod selector",
			args:  []string{"drain", "node-a", "--pod-selector=app in ("},
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

func TestDrainRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"drain", "node-a", "--grace-period=1", "--grace-period=2"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("drain").Flag("--grace-period").Value().Text("2"),
	)
}

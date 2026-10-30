package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestAutoscaleDeclaresFrontendSelectionAndBounds(t *testing.T) {
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
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "invalid output format",
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--output=wat"},
			ready: false,
		},
		{
			name:  "resource and max",
			args:  []string{"autoscale", "deployment/foo", "--max=3"},
			ready: true,
		},
		{
			name:  "assigned short filename",
			args:  []string{"autoscale", "-f=deployment.yaml", "--max=3"},
			ready: true,
		},
		{
			name:  "filename and max",
			args:  []string{"autoscale", "-f", "deployment.yaml", "--max=3"},
			ready: true,
		},
		{
			name:  "kustomize and max",
			args:  []string{"autoscale", "-k", "dir", "--max=3"},
			ready: true,
		},
		{
			name:  "missing max",
			args:  []string{"autoscale", "deployment/foo"},
			ready: false,
		},
		{
			name:  "max below one",
			args:  []string{"autoscale", "deployment/foo", "--max=0"},
			ready: false,
		},
		{
			name:  "min above max",
			args:  []string{"autoscale", "deployment/foo", "--min=4", "--max=3"},
			ready: false,
		},
		{
			name:  "cpu forms conflict",
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--cpu=500m", "--cpu-percent=60"},
			ready: false,
		},
		{
			name:  "invalid cpu target",
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--cpu=watts"},
			ready: false,
		},
		{
			name:  "invalid memory target",
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--memory=watts"},
			ready: false,
		},
		{
			name:  "missing source",
			args:  []string{"autoscale", "--max=3"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"autoscale", "--max=3", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"autoscale", "--max=3", "-k", "dir", "--recursive"},
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

func TestAutoscaleRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"autoscale", "deployment/foo", "--max=2", "--max=3"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("autoscale").Flag("--max").Value().Text("3"),
	)
}

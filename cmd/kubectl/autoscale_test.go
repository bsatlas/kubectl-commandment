package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestAutoscaleDeclaresFrontendSelectionAndBounds(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "resource and max",
			args:  []string{"autoscale", "deployment/foo", "--max=3"},
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
			name:  "explicit empty cpu",
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--cpu="},
			ready: true,
		},
		{
			name:  "cpu percent with explicit empty cpu",
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--cpu-percent=60", "--cpu="},
			ready: true,
		},
		{
			name:  "explicit empty memory",
			args:  []string{"autoscale", "deployment/foo", "--max=3", "--memory="},
			ready: true,
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
			name:  "filename and explicit empty kustomize",
			args:  []string{"autoscale", "--max=3", "-f", "a.yaml", "--kustomize="},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"autoscale", "--max=3", "--kustomize="},
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

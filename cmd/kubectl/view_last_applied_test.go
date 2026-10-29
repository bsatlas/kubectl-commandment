package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestApplyViewLastAppliedDeclaresSourceAndOutputDomain(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "resource source",
			args: []string{"apply", "view-last-applied", "deployment/nginx"},
		},
		{
			name: "filename source",
			args: []string{"apply", "view-last-applied", "-f", "object.yaml", "-o", "json"},
		},
		{
			name: "kustomize source",
			args: []string{"apply", "view-last-applied", "-k", "overlay"},
		},
		{
			name:     "source required",
			args:     []string{"apply", "view-last-applied"},
			rejected: true,
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"apply", "view-last-applied", "-f", "object.yaml", "-k", "overlay"},
			rejected: true,
		},
		{
			name:     "recursive kustomize conflict",
			args:     []string{"apply", "view-last-applied", "-k", "overlay", "-R"},
			rejected: true,
		},
		{
			name:     "invalid output",
			args:     []string{"apply", "view-last-applied", "deployment/nginx", "-o", "name"},
			rejected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if !test.rejected {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t, clitest.InspectionRejected(), clitest.FailsWith(clitest.DiagnosticAt(0).Present()))
		})
	}
}

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetResourcesDeclaresSourceAndResourceListSemantics(t *testing.T) {
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
			name: "limits update",
			args: []string{"set", "resources", "deployment/nginx", "--limits=cpu=200m,memory=512Mi"},
		},
		{
			name: "requests update",
			args: []string{"set", "resources", "deployment/nginx", "--requests=cpu=100m"},
		},
		{
			name:     "update required",
			args:     []string{"set", "resources", "deployment/nginx"},
			rejected: true,
		},
		{
			name:     "invalid resource list",
			args:     []string{"set", "resources", "deployment/nginx", "--limits=cpu"},
			rejected: true,
		},
		{
			name:     "source required",
			args:     []string{"set", "resources", "--limits=cpu=200m"},
			rejected: true,
		},
		{
			name: "kustomize source",
			args: []string{"set", "resources", "-k", "overlay", "--limits=cpu=200m"},
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"set", "resources", "-f", "object.yaml", "-k", "overlay", "--limits=cpu=200m"},
			rejected: true,
		},
		{
			name:     "recursive kustomize conflict",
			args:     []string{"set", "resources", "deployment/nginx", "-k", "overlay", "-R", "--limits=cpu=200m"},
			rejected: true,
		},
		{
			name:     "all conflicts with selector",
			args:     []string{"set", "resources", "deployment", "--limits=cpu=200m", "--all", "--selector=app=web"},
			rejected: true,
		},
		{
			name:     "local forbids resource arguments",
			args:     []string{"set", "resources", "deployment/nginx", "--limits=cpu=200m", "--local"},
			rejected: true,
		},
		{
			name: "local filename source",
			args: []string{"set", "resources", "-f", "object.yaml", "--limits=cpu=200m", "--local"},
		},
		{
			name:     "local conflicts with server dry run",
			args:     []string{"set", "resources", "-f", "object.yaml", "--limits=cpu=200m", "--local", "--dry-run=server"},
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

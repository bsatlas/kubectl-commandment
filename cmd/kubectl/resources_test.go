package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetResourcesDeclaresSourceAndResourceListSemantics(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
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
			name:     "local forbids resource arguments",
			args:     []string{"set", "resources", "deployment/nginx", "--limits=cpu=200m", "--local"},
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

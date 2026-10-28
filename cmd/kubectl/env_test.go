package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetEnvDeclaresResourceAndEnvironmentRelationships(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "resource and assignment",
			args: []string{"set", "env", "deployment/nginx", "MODE=prod"},
		},
		{
			name: "filename and assignment",
			args: []string{"set", "env", "-f", "object.yaml", "MODE=prod"},
		},
		{
			name: "multiple environment changes",
			args: []string{"set", "env", "deployment/nginx", "MODE=prod", "DEBUG=true"},
		},
		{
			name:     "resource after environment change is rejected",
			args:     []string{"set", "env", "deployment/nginx", "MODE=prod", "service/backend"},
			rejected: true,
		},
		{
			name:     "kustomize alone preserves upstream source quirk",
			args:     []string{"set", "env", "-k", "overlay", "MODE=prod"},
			rejected: true,
		},
		{
			name:     "keys require from",
			args:     []string{"set", "env", "deployment/nginx", "--keys=MODE"},
			rejected: true,
		},
		{
			name:     "set and remove conflict",
			args:     []string{"set", "env", "deployment/nginx", "MODE=prod", "MODE-"},
			rejected: true,
		},
		{
			name:     "local forbids resource arguments",
			args:     []string{"set", "env", "deployment/nginx", "MODE=prod", "--local"},
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

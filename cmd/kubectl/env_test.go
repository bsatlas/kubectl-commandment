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
			name: "keys with from",
			args: []string{"set", "env", "deployment/nginx", "--keys=MODE", "--from=configmap/app"},
		},
		{
			name:     "set and remove conflict",
			args:     []string{"set", "env", "deployment/nginx", "MODE=prod", "MODE-"},
			rejected: true,
		},
		{
			name:     "source required",
			args:     []string{"set", "env", "MODE=prod"},
			rejected: true,
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"set", "env", "-f", "object.yaml", "-k", "overlay", "MODE=prod"},
			rejected: true,
		},
		{
			name:     "recursive kustomize conflict",
			args:     []string{"set", "env", "deployment/nginx", "MODE=prod", "-k", "overlay", "-R"},
			rejected: true,
		},
		{
			name:     "all conflicts with selector",
			args:     []string{"set", "env", "deployment", "MODE=prod", "--all", "--selector=app=web"},
			rejected: true,
		},
		{
			name: "list without output",
			args: []string{"set", "env", "deployment/nginx", "--list"},
		},
		{
			name:     "list conflicts with output",
			args:     []string{"set", "env", "deployment/nginx", "--list", "-o", "yaml"},
			rejected: true,
		},
		{
			name:     "local forbids resource arguments",
			args:     []string{"set", "env", "deployment/nginx", "MODE=prod", "--local"},
			rejected: true,
		},
		{
			name: "local filename source",
			args: []string{"set", "env", "-f", "object.yaml", "MODE=prod", "--local"},
		},
		{
			name:     "local conflicts with server dry run",
			args:     []string{"set", "env", "-f", "object.yaml", "MODE=prod", "--local", "--dry-run=server"},
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

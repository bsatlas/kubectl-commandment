package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetServiceAccountDeclaresSourceWithoutInventingLocalResourcePolicy(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "resource source",
			args: []string{"set", "serviceaccount", "deployment/nginx", "builder"},
		},
		{
			name: "filename source",
			args: []string{"set", "serviceaccount", "-f", "object.yaml", "builder"},
		},
		{
			name:     "source required",
			args:     []string{"set", "serviceaccount", "builder"},
			rejected: true,
		},
		{
			name: "local resource arguments remain admitted",
			args: []string{"set", "serviceaccount", "deployment/nginx", "builder", "--local"},
		},
		{
			name:     "local server dry run conflicts",
			args:     []string{"set", "serviceaccount", "-f", "object.yaml", "builder", "--local", "--dry-run=server"},
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

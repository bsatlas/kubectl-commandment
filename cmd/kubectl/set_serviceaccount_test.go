package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetServiceAccountDeclaresSourceWithoutInventingLocalResourcePolicy(t *testing.T) {
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
			args: []string{"set", "serviceaccount", "deployment/nginx", "builder"},
		},
		{
			name: "filename source",
			args: []string{"set", "serviceaccount", "-f", "object.yaml", "builder"},
		},
		{
			name: "kustomize source",
			args: []string{"set", "serviceaccount", "-k", "overlay", "builder"},
		},
		{
			name:     "source required",
			args:     []string{"set", "serviceaccount", "builder"},
			rejected: true,
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"set", "serviceaccount", "-f", "object.yaml", "-k", "overlay", "builder"},
			rejected: true,
		},
		{
			name:     "recursive kustomize conflict",
			args:     []string{"set", "serviceaccount", "deployment/nginx", "builder", "-k", "overlay", "-R"},
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
		{
			name: "assigned short filename",
			args: []string{"set", "serviceaccount", "-f=object.yaml", "builder"},
		},
		{
			name: "repeated field manager keeps last",
			args: []string{"set", "serviceaccount", "deployment/nginx", "builder", "--field-manager=first", "--field-manager=second"},
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

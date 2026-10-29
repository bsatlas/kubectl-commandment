package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestEditDeclaresResourceSourceGrammar(t *testing.T) {
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
			name:  "resource argument",
			args:  []string{"edit", "deployment/foo"},
			ready: true,
		},
		{
			name:  "filename",
			args:  []string{"edit", "-f", "deployment.yaml"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"edit"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"edit", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"edit", "-f", "a.yaml", "--kustomize="},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"edit", "--kustomize="},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"edit", "-k", "dir", "--recursive"},
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

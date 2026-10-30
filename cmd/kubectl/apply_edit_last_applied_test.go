package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestEditLastAppliedDeclaresResourceSourceGrammar(t *testing.T) {
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
			args:  []string{"apply", "edit-last-applied", "deployment/foo"},
			ready: true,
		},
		{
			name:  "filename",
			args:  []string{"apply", "edit-last-applied", "-f", "deployment.yaml"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"apply", "edit-last-applied"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"apply", "edit-last-applied", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"apply", "edit-last-applied", "-k", "dir", "--recursive"},
			ready: false,
		},
		{
			name:  "assigned short filename",
			args:  []string{"apply", "edit-last-applied", "-f=object.yaml"},
			ready: true,
		},
		{
			name:  "repeated field manager keeps last",
			args:  []string{"apply", "edit-last-applied", "deployment/nginx", "--field-manager=first", "--field-manager=second"},
			ready: true,
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

package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestExposeDeclaresResourceSourceGrammar(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
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
			args:  []string{"expose", "deployment/foo"},
			ready: true,
		},
		{
			name:  "filename",
			args:  []string{"expose", "-f", "deployment.yaml"},
			ready: true,
		},
		{
			name:  "kustomize",
			args:  []string{"expose", "-k", "dir"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"expose"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"expose", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"expose", "-f", "a.yaml", "--kustomize="},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"expose", "--kustomize="},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"expose", "-k", "dir", "--recursive"},
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

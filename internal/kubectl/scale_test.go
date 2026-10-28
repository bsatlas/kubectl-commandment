package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestScaleDeclaresSelectionAndReplicaBounds(t *testing.T) {
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
			name:  "resource replicas",
			args:  []string{"scale", "deployment/foo", "--replicas=2"},
			ready: true,
		},
		{
			name:  "file replicas",
			args:  []string{"scale", "-f", "deployment.yaml", "--replicas=2"},
			ready: true,
		},
		{
			name:  "kustomize replicas",
			args:  []string{"scale", "-k", "dir", "--replicas=2"},
			ready: true,
		},
		{
			name:  "missing replicas",
			args:  []string{"scale", "deployment/foo"},
			ready: false,
		},
		{
			name:  "negative replicas",
			args:  []string{"scale", "deployment/foo", "--replicas=-1"},
			ready: false,
		},
		{
			name:  "invalid current replicas",
			args:  []string{"scale", "deployment/foo", "--replicas=2", "--current-replicas=-2"},
			ready: false,
		},
		{
			name:  "missing source",
			args:  []string{"scale", "--replicas=2"},
			ready: false,
		},
		{
			name:  "all with selector follows upstream builder semantics",
			args:  []string{"scale", "deployment", "--replicas=2", "--all", "--selector=app=web"},
			ready: true,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"scale", "--replicas=2", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"scale", "--replicas=2", "-f", "a.yaml", "--kustomize="},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"scale", "--replicas=2", "--kustomize="},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"scale", "-k", "dir", "--recursive", "--replicas=2"},
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

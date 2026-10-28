package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestDrainDeclaresNodeSelectionAndPodSelectorLanguage(t *testing.T) {
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
			name:  "node argument",
			args:  []string{"drain", "node-a"},
			ready: true,
		},
		{
			name:  "node selector",
			args:  []string{"drain", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "missing selection",
			args:  []string{"drain"},
			ready: false,
		},
		{
			name:  "argument and selector",
			args:  []string{"drain", "node-a", "--selector=role=worker"},
			ready: false,
		},
		{
			name:  "argument and explicit empty selector",
			args:  []string{"drain", "node-a", "--selector="},
			ready: true,
		},
		{
			name:  "explicit empty selector",
			args:  []string{"drain", "--selector="},
			ready: true,
		},
		{
			name:  "valid pod selector",
			args:  []string{"drain", "node-a", "--pod-selector=app=web"},
			ready: true,
		},
		{
			name:  "invalid pod selector",
			args:  []string{"drain", "node-a", "--pod-selector=app in ("},
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

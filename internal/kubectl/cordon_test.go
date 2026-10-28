package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestCordonDeclaresNodeSelectionGrammar(t *testing.T) {
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
			args:  []string{"cordon", "node-a"},
			ready: true,
		},
		{
			name:  "selector",
			args:  []string{"cordon", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "missing selection",
			args:  []string{"cordon"},
			ready: false,
		},
		{
			name:  "argument and selector",
			args:  []string{"cordon", "node-a", "--selector=role=worker"},
			ready: false,
		},
		{
			name:  "argument and explicit empty selector",
			args:  []string{"cordon", "node-a", "--selector="},
			ready: true,
		},
		{
			name:  "explicit empty selector",
			args:  []string{"cordon", "--selector="},
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

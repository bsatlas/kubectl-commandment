package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestUncordonDeclaresNodeSelectionGrammar(t *testing.T) {
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
			args:  []string{"uncordon", "node-a"},
			ready: true,
		},
		{
			name:  "selector",
			args:  []string{"uncordon", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "missing selection",
			args:  []string{"uncordon"},
			ready: false,
		},
		{
			name:  "argument and selector",
			args:  []string{"uncordon", "node-a", "--selector=role=worker"},
			ready: false,
		},
		{
			name:  "argument and explicit empty selector",
			args:  []string{"uncordon", "node-a", "--selector="},
			ready: true,
		},
		{
			name:  "explicit empty selector",
			args:  []string{"uncordon", "--selector="},
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

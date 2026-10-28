package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestConfigSetContextCurrentUsesBooleanValue(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name:     "explicit false still requires name",
			args:     []string{"config", "set-context", "--current=false"},
			rejected: true,
		},
		{
			name: "name with explicit false",
			args: []string{"config", "set-context", "named", "--current=false"},
		},
		{
			name:     "name with true current",
			args:     []string{"config", "set-context", "named", "--current=true"},
			rejected: true,
		},
		{
			name: "true current without name",
			args: []string{"config", "set-context", "--current=true"},
		},
		{
			name: "empty name with true current",
			args: []string{"config", "set-context", "", "--current=true"},
		},
		{
			name:     "empty name with false current",
			args:     []string{"config", "set-context", "", "--current=false"},
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
			inspection.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestConfigSetContextTreatsKnownNamesAsCompletionCandidates(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	inspection := cli.InspectionRequest{
		Args: []string{"--kubeconfig", path, "config", "set-context", "missing"},
	}
	clitest.Inspect(t.Context(), app, inspection).Require(t, clitest.InspectionReady())

	completion := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "set-context", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	}
	clitest.Complete(t.Context(), app, completion).Require(t, clitest.CompletionItem("dev").Present())
}

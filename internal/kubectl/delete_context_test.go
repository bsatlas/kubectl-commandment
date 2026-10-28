package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestConfigDeleteContextTreatsKnownNamesAsCompletionCandidates(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	inspectionRequest := cli.InspectionRequest{
		Args: []string{"--kubeconfig", path, "config", "delete-context", "missing"},
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.InspectionReady())

	completionRequest := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "delete-context", "d"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     1,
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItem("dev").Present())
}

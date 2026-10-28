package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestConfigDeleteUserTreatsKnownNamesAsCompletionCandidates(t *testing.T) {
	t.Parallel()

	path := writeConfigTestKubeconfig(t, "dev", "dev")
	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}

	inspectionRequest := cli.InspectionRequest{
		Args: []string{"--kubeconfig", path, "config", "delete-user", "missing"},
	}
	clitest.Inspect(t.Context(), app, inspectionRequest).Require(t, clitest.InspectionReady())

	completionRequest := cli.CompletionRequest{
		Args: []string{"--kubeconfig", path, "config", "delete-user", "us"},
		Cursor: cli.CompletionCursor{
			Argument: 4,
			Byte:     2,
		},
	}
	clitest.Complete(t.Context(), app, completionRequest).Require(t, clitest.CompletionItem("user").Present())
}

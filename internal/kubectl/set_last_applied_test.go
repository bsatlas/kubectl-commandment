package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestApplySetLastAppliedRequiresFilename(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	readyRequests := []cli.InspectionRequest{
		{
			Args: []string{"apply", "set-last-applied", "-f", "object.yaml"},
		},
		{
			Args: []string{"apply", "set-last-applied", "-f", "one.yaml", "-f", "two.yaml"},
		},
	}
	for _, request := range readyRequests {
		clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
	}

	rejectedRequest := cli.InspectionRequest{
		Args: []string{"apply", "set-last-applied"},
	}
	clitest.Inspect(t.Context(), app, rejectedRequest).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
	)
}

package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestRolloutUndoDryRunDomain(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	cases := []struct {
		name string
		args []string
		want clitest.InspectionExpectation
	}{
		{
			name: "none",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=none"},
			want: clitest.InspectionReady(),
		},
		{
			name: "client",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=client"},
			want: clitest.InspectionReady(),
		},
		{
			name: "server",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=server"},
			want: clitest.InspectionReady(),
		},
		{
			name: "legacy true",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=true"},
			want: clitest.InspectionReady(),
		},
		{
			name: "legacy false",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=false"},
			want: clitest.InspectionReady(),
		},
		{
			name: "unknown",
			args: []string{"rollout", "undo", "deployment/frontend", "--dry-run=remote"},
			want: clitest.InspectionRejected(),
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			proof := clitest.Inspect(t.Context(), app, request)
			proof.Require(t, test.want)
		})
	}
}

func TestRolloutUndoRejectsNegativeRevision(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"rollout", "undo", "deployment/frontend", "--to-revision", "-1"},
	})
	proof.Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.Diagnostic(cli.DiagnosticRuleViolation).Present()),
	)
}

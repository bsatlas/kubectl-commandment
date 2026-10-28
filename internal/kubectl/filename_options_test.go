package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestRolloutFilenameOptions(t *testing.T) {
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
			name: "CSV filenames",
			args: []string{"rollout", "history", "--filename", `one.yaml,"two,three.yaml"`},
			want: clitest.InspectionReady(),
		},
		{
			name: "short joined filename",
			args: []string{"rollout", "history", "-fone.yaml"},
			want: clitest.InspectionReady(),
		},
		{
			name: "short assigned filename",
			args: []string{"rollout", "history", "-f=one.yaml"},
			want: clitest.InspectionReady(),
		},
		{
			name: "filename and kustomize",
			args: []string{"rollout", "history", "--filename", "one.yaml", "--kustomize", "overlay"},
			want: clitest.InspectionRejected(),
		},
		{
			name: "kustomize and recursive",
			args: []string{"rollout", "history", "--kustomize", "overlay", "--recursive"},
			want: clitest.InspectionRejected(),
		},
		{
			name: "source required",
			args: []string{"rollout", "history"},
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

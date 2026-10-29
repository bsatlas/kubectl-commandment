package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRolloutResourceGrammar(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	cases := []struct {
		name string
		args []string
		want clitest.InspectionExpectation
	}{
		{
			name: "type and names",
			args: []string{"rollout", "history", "deployment", "frontend", "backend"},
			want: clitest.InspectionReady(),
		},
		{
			name: "qualified resources",
			args: []string{"rollout", "history", "deployment/frontend", "daemonset/agent"},
			want: clitest.InspectionReady(),
		},
		{
			name: "mixed forms",
			args: []string{"rollout", "history", "deployment", "daemonset/agent"},
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

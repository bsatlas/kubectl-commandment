package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRolloutPrintOutputGrammar(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	cases := []struct {
		name string
		args []string
		want clitest.InspectionExpectation
	}{
		{
			name: "canonical format",
			args: []string{"rollout", "history", "deployment/frontend", "--output", "json"},
			want: clitest.InspectionReady(),
		},
		{
			name: "short assigned format",
			args: []string{"rollout", "history", "deployment/frontend", "-o=json"},
			want: clitest.InspectionReady(),
		},
		{
			name: "inline template",
			args: []string{"rollout", "history", "deployment/frontend", "--output", `go-template={{.metadata.name}}`},
			want: clitest.InspectionReady(),
		},
		{
			name: "unknown format",
			args: []string{"rollout", "history", "deployment/frontend", "--output", "table"},
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

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestExecFrontendContract(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"exec", "pod-a", "--", "sh", "--container=remote"}}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(1).Role(cli.TokenRolePositional),
		clitest.TokenAt(2).Role(cli.TokenRoleEndOfOptions),
		clitest.TokenAt(3).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(4).Role(cli.TokenRolePassThrough),
	)

	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{name: "filename CSV quoting", args: []string{"exec", "--filename=\"pod,copy.yaml\"", "--", "sh"}, ready: true},
		{name: "joined container shorthand", args: []string{"exec", "pod-a", "-csidecar", "--", "sh"}, ready: true},
		{name: "assigned filename shorthand", args: []string{"exec", "-f=pod.yaml", "--", "sh"}, ready: true},
		{name: "missing separator", args: []string{"exec", "pod-a", "sh"}},
		{name: "missing resource", args: []string{"exec", "--", "sh"}},
		{name: "missing command", args: []string{"exec", "pod-a", "--"}},
		{name: "nonpositive timeout", args: []string{"exec", "pod-a", "--pod-running-timeout=0s", "--", "sh"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			inspection := clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: test.args})
			if test.ready {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t, clitest.InspectionRejected())
		})
	}
}

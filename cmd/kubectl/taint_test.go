package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestTaintDeclaresResourceAndChangeGrammar(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "named node",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule"},
			ready: true,
		},
		{
			name:  "selector nodes",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule", "--selector=role=worker"},
			ready: true,
		},
		{
			name:  "all nodes",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule", "--all"},
			ready: true,
		},
		{
			name:  "missing update",
			args:  []string{"taint", "nodes", "node-a"},
			ready: false,
		},
		{
			name:  "missing target",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule"},
			ready: false,
		},
		{
			name:  "all with selector",
			args:  []string{"taint", "nodes", "dedicated=special:NoSchedule", "--all", "--selector=role=worker"},
			ready: false,
		},
		{
			name:  "add and remove same taint",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule", "dedicated:NoSchedule-"},
			ready: false,
		},
		{
			name:  "invalid taint key",
			args:  []string{"taint", "nodes", "node-a", "bad$key=value:NoSchedule"},
			ready: false,
		},
		{
			name:  "invalid taint value",
			args:  []string{"taint", "nodes", "node-a", "dedicated=bad$value:NoSchedule"},
			ready: false,
		},
		{
			name:  "invalid effect",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:Sometimes"},
			ready: false,
		},
		{
			name:  "remove without effect treats equals as invalid key syntax",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special-"},
			ready: false,
		},
		{
			name:  "remove with value and effect follows upstream parser",
			args:  []string{"taint", "nodes", "node-a", "dedicated=special:NoSchedule-"},
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

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestLogsFrontendContract(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{name: "resource", args: []string{"logs", "pod-a"}, ready: true},
		{name: "selector", args: []string{"logs", "-lapp=demo"}, ready: true},
		{name: "joined container shorthand", args: []string{"logs", "pod-a", "-csidecar"}, ready: true},
		{name: "last tail wins", args: []string{"logs", "pod-a", "--tail=-2", "--tail=5"}, ready: true},
		{name: "missing source", args: []string{"logs"}},
		{name: "resource and selector", args: []string{"logs", "pod-a", "--selector=app=demo"}},
		{name: "inline and flag container", args: []string{"logs", "pod-a", "inline", "--container=flagged"}},
		{name: "all and named container", args: []string{"logs", "pod-a", "--all-containers", "--container=flagged"}},
		{name: "since and since time", args: []string{"logs", "pod-a", "--since=1m", "--since-time=2026-10-28T12:00:00Z"}},
		{name: "invalid tail", args: []string{"logs", "pod-a", "--tail=-2"}},
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

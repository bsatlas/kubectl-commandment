package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestAttachFrontendContract(t *testing.T) {
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
		{name: "pod", args: []string{"attach", "pod-a"}, ready: true},
		{name: "type and name", args: []string{"attach", "pod", "pod-a"}, ready: true},
		{name: "joined container shorthand", args: []string{"attach", "pod-a", "-csidecar"}, ready: true},
		{name: "assigned container shorthand", args: []string{"attach", "pod-a", "-c=sidecar"}, ready: true},
		{name: "last timeout wins", args: []string{"attach", "pod-a", "--pod-running-timeout=0s", "--pod-running-timeout=1s"}, ready: true},
		{name: "missing resource", args: []string{"attach"}},
		{name: "too many arguments", args: []string{"attach", "pod", "pod-a", "extra"}},
		{name: "nonpositive timeout", args: []string{"attach", "pod-a", "--pod-running-timeout=0s"}},
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

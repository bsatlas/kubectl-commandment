package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCopyFrontendContract(t *testing.T) {
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
		{name: "local to remote", args: []string{"cp", "./local", "pod-a:/remote"}, ready: true},
		{name: "remote to local", args: []string{"cp", "ns/pod-a:/remote", "./local"}, ready: true},
		{name: "joined container shorthand", args: []string{"cp", "./local", "pod-a:/remote", "-csidecar"}, ready: true},
		{name: "assigned container shorthand", args: []string{"cp", "./local", "pod-a:/remote", "-c=sidecar"}, ready: true},
		{name: "local to local", args: []string{"cp", "./left", "./right"}},
		{name: "remote to remote", args: []string{"cp", "pod-a:/left", "pod-b:/right"}},
		{name: "empty remote path", args: []string{"cp", "./local", "pod-a:"}},
		{name: "missing destination", args: []string{"cp", "./local"}},
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

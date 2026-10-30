package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRunFrontendContract(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"run", "work", "--image=busybox", "--", "arg", "--image=remote"}}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(1).Role(cli.TokenRolePositional),
		clitest.TokenAt(3).Role(cli.TokenRoleEndOfOptions),
		clitest.TokenAt(4).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(5).Role(cli.TokenRolePassThrough),
	)

	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{name: "dry run client", args: []string{"run", "work", "--image=busybox", "--dry-run=client"}, ready: true},
		{name: "legacy dry run true", args: []string{"run", "work", "--image=busybox", "--dry-run=true"}, ready: true},
		{name: "legacy dry run false", args: []string{"run", "work", "--image=busybox", "--dry-run=false"}, ready: true},
		{name: "filename CSV quoting", args: []string{"run", "work", "--image=busybox", "--filename=\"pod,copy.yaml\""}, ready: true},
		{name: "joined label shorthand", args: []string{"run", "work", "--image=busybox", "-lapp=demo"}, ready: true},
		{name: "embedded equals in env and annotation", args: []string{"run", "work", "--image=busybox", "--env=TOKEN=a=b", "--annotations=owner=a=b"}, ready: true},
		{name: "missing image", args: []string{"run", "work"}},
		{name: "tty requires stdin", args: []string{"run", "work", "--image=busybox", "--tty"}},
		{name: "expose requires port", args: []string{"run", "work", "--image=busybox", "--expose"}},
		{name: "rm requires attach", args: []string{"run", "work", "--image=busybox", "--rm"}},
		{name: "attach conflicts with dry run", args: []string{"run", "work", "--image=busybox", "--attach", "--dry-run=client"}},
		{name: "invalid dry run", args: []string{"run", "work", "--image=busybox", "--dry-run=bogus"}},
		{name: "invalid port", args: []string{"run", "work", "--image=busybox", "--port=not-a-number"}},
		{name: "name after separator", args: []string{"run", "--", "work"}},
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

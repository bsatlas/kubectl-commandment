package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestDebugFrontendContract(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"debug", "pod-a", "pod-b", "--", "sh", "--image=busybox"}}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(1).Role(cli.TokenRolePositional),
		clitest.TokenAt(2).Role(cli.TokenRolePositional),
		clitest.TokenAt(3).Role(cli.TokenRoleEndOfOptions),
		clitest.TokenAt(4).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(5).Role(cli.TokenRolePassThrough),
	)

	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{name: "target and image", args: []string{"debug", "pod-a", "--image=busybox"}, ready: true},
		{name: "filename CSV quoting", args: []string{"debug", "--filename=\"pod,copy.yaml\"", "--image=busybox"}, ready: true},
		{name: "joined container shorthand", args: []string{"debug", "pod-a", "--image=busybox", "-cdebugger"}, ready: true},
		{name: "copy with image replacement", args: []string{"debug", "pod-a", "--copy-to=pod-copy", "--set-image=*=busybox"}, ready: true},
		{name: "missing image", args: []string{"debug", "pod-a"}},
		{name: "copy without image or replacement", args: []string{"debug", "pod-a", "--copy-to=pod-copy"}},
		{name: "copy target conflicts", args: []string{"debug", "pod-a", "--copy-to=pod-copy", "--target=main", "--image=busybox"}},
		{name: "tty requires stdin", args: []string{"debug", "pod-a", "--image=busybox", "--tty"}},
		{name: "invalid profile", args: []string{"debug", "pod-a", "--image=busybox", "--profile=unknown"}},
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

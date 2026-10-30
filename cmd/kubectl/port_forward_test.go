package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestPortForwardFrontendContract(t *testing.T) {
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
		{name: "numeric port", args: []string{"port-forward", "pod/pod-a", "5000"}, ready: true},
		{name: "random local port", args: []string{"port-forward", "pod/pod-a", ":5000"}, ready: true},
		{name: "numeric to named", args: []string{"port-forward", "service/service-a", "8443:https"}, ready: true},
		{name: "named remote", args: []string{"port-forward", "pod/pod-a", "http"}, ready: true},
		{name: "named to named", args: []string{"port-forward", "pod/pod-a", "http:http"}, ready: true},
		{name: "missing port", args: []string{"port-forward", "pod/pod-a"}},
		{name: "missing remote port", args: []string{"port-forward", "pod/pod-a", "5000:"}},
		{name: "too many port components", args: []string{"port-forward", "pod/pod-a", "5000:6000:7000"}},
		{name: "named local port", args: []string{"port-forward", "pod/pod-a", "local:http"}},
		{name: "invalid address", args: []string{"port-forward", "pod/pod-a", "5000", "--address=not-an-ip"}},
		{name: "nonpositive timeout", args: []string{"port-forward", "pod/pod-a", "5000", "--pod-running-timeout=0s"}},
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

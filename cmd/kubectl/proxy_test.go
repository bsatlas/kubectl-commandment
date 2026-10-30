package main

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestProxyFrontendContract(t *testing.T) {
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
		{name: "defaults", args: []string{"proxy"}, ready: true},
		{name: "ignored positional arguments", args: []string{"proxy", "ignored"}, ready: true},
		{name: "unix socket with default port", args: []string{"proxy", "-u/tmp/kubectl.sock"}, ready: true},
		{name: "random port shorthand", args: []string{"proxy", "-p0"}, ready: true},
		{name: "unix socket with custom port", args: []string{"proxy", "--unix-socket=/tmp/kubectl.sock", "--port=9000"}},
		{name: "invalid host filter", args: []string{"proxy", "--accept-hosts=["}},
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

func TestProxyStopsOnInvocationCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cfg := proxyConfig{
		Client: clientConfig{
			Server:   "https://127.0.0.1:1",
			Insecure: true,
		},
		Streams: genericiooptions.IOStreams{
			Out:    io.Discard,
			ErrOut: io.Discard,
		},
		Address:       "127.0.0.1",
		APIPrefix:     "/",
		DisableFilter: true,
		Port:          0,
		WwwPrefix:     "/static/",
	}
	done := make(chan error, 1)
	go func() {
		done <- runProxy(ctx, &cfg)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("runProxy() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runProxy() did not stop after invocation cancellation")
	}
}

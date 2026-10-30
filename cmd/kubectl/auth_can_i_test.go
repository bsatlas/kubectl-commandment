package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestAuthCanIFrontendPolicy(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "resource check",
			args: []string{"auth", "can-i", "get", "pods"},
		},
		{
			name: "non-resource check",
			args: []string{"auth", "can-i", "get", "/logs/"},
		},
		{
			name: "unknown verb remains admissible for runtime warning",
			args: []string{"auth", "can-i", "frobnicate", "pods"},
		},
		{
			name: "list without operands",
			args: []string{"auth", "can-i", "--list"},
		},
		{
			name:     "missing target",
			args:     []string{"auth", "can-i", "get"},
			rejected: true,
		},
		{
			name:     "list with operands",
			args:     []string{"auth", "can-i", "--list", "get", "pods"},
			rejected: true,
		},
		{
			name:     "list with quiet",
			args:     []string{"auth", "can-i", "--list", "--quiet"},
			rejected: true,
		},
		{
			name:     "list with all namespaces",
			args:     []string{"auth", "can-i", "--list", "--all-namespaces"},
			rejected: true,
		},
		{
			name:     "list with subresource",
			args:     []string{"auth", "can-i", "--list", "--subresource=log"},
			rejected: true,
		},
		{
			name:     "no headers without list",
			args:     []string{"auth", "can-i", "get", "pods", "--no-headers"},
			rejected: true,
		},
		{
			name: "no headers with list",
			args: []string{"auth", "can-i", "--list", "--no-headers"},
		},
		{
			name:     "non-resource subresource",
			args:     []string{"auth", "can-i", "get", "/logs/", "--subresource=tail"},
			rejected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: test.args})
			if !test.rejected {
				proof.Require(t, clitest.InspectionReady())
				return
			}
			proof.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestAuthCanIUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	tests := []struct {
		name string
		args []string
		flag string
		want string
	}{
		{
			name: "all namespaces",
			args: []string{"auth", "can-i", "get", "pods", "--all-namespaces", "--all-namespaces=false"},
			flag: "--all-namespaces",
			want: "false",
		},
		{
			name: "list",
			args: []string{"auth", "can-i", "get", "pods", "--list", "--list=false"},
			flag: "--list",
			want: "false",
		},
		{
			name: "no headers",
			args: []string{"auth", "can-i", "get", "pods", "--no-headers", "--no-headers=false"},
			flag: "--no-headers",
			want: "false",
		},
		{
			name: "quiet",
			args: []string{"auth", "can-i", "--list", "--quiet", "--quiet=false"},
			flag: "--quiet",
			want: "false",
		},
		{
			name: "subresource",
			args: []string{"auth", "can-i", "get", "pods", "--subresource=log", "--subresource=scale"},
			flag: "--subresource",
			want: "scale",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value := clitest.Command("auth", "can-i").Flag(test.flag).Value()
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: test.args}).Require(t,
				clitest.InspectionReady(),
				value.Resolved(),
				value.Text(test.want),
			)
		})
	}
}

func TestAuthCanIPropagatesInvocationCancellation(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews" {
			http.Error(writer, "unexpected request: "+request.Method+" "+request.URL.Path, http.StatusNotFound)
			return
		}
		close(started)
		<-release
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"kind":"SelfSubjectAccessReview","apiVersion":"authorization.k8s.io/v1","status":{"allowed":true}}`))
	}))
	t.Cleanup(server.Close)

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	request := cli.RunRequest{
		Args:   []string{"--server", server.URL, "auth", "can-i", "get", "/healthz"},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(ctx, request)
	}()

	<-started
	cancel()
	select {
	case err := <-errCh:
		close(release)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		<-errCh
		t.Fatal("Run() did not return after invocation cancellation")
	}
}

func TestAuthCanIRetainsDynamicUnknownVerbWarning(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews" {
			http.Error(writer, "unexpected request: "+request.Method+" "+request.URL.Path, http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"kind":"SelfSubjectAccessReview","apiVersion":"authorization.k8s.io/v1","status":{"allowed":true}}`))
	}))
	t.Cleanup(server.Close)

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	request := cli.RunRequest{
		Args:   []string{"--server", server.URL, "auth", "can-i", "frobnicate", "/logs/", "--all-namespaces"},
		Stdout: &stdout,
		Stderr: &stderr,
	}

	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := stdout.String(), "yes\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "Warning: verb 'frobnicate' is not a known verb\n\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

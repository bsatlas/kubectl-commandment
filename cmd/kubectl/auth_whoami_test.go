package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestAuthWhoAmIAdmitsIgnoredArguments(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"auth", "whoami", "ignored", "arguments"},
	}).Require(t, clitest.InspectionReady())
}

func TestAuthWhoAmIIgnoresArgumentsAtRuntime(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/apis/authentication.k8s.io/v1/selfsubjectreviews" {
			http.Error(writer, "unexpected request: "+request.Method+" "+request.URL.Path, http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"SelfSubjectReview","status":{"userInfo":{"username":"alice"}}}`))
	}))
	t.Cleanup(server.Close)

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	request := cli.RunRequest{
		Args:   []string{"--server", server.URL, "auth", "whoami", "ignored", "arguments"},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	if err := app.Run(t.Context(), request); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

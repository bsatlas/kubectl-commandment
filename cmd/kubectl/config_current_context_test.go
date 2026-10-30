package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestConfigCurrentContextAdmitsIgnoredArguments(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"config", "current-context", "ignored", "arguments"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

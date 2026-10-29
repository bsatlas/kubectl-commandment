package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetSelectorDeclaresSourceAndSelectorGrammar(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "resource and selector",
			args: []string{"set", "selector", "service/frontend", "app=frontend"},
		},
		{
			name: "filename and selector",
			args: []string{"set", "selector", "-f", "service.yaml", "app=frontend"},
		},
		{
			name:     "source required",
			args:     []string{"set", "selector", "app=frontend"},
			rejected: true,
		},
		{
			name:     "selector required",
			args:     []string{"set", "selector", "service/frontend"},
			rejected: true,
		},
		{
			name:     "invalid selector",
			args:     []string{"set", "selector", "service/frontend", "app in ("},
			rejected: true,
		},
		{
			name:     "local forbids resource arguments",
			args:     []string{"set", "selector", "service/frontend", "app=frontend", "--local"},
			rejected: true,
		},
		{
			name: "local filename source",
			args: []string{"set", "selector", "-f", "service.yaml", "app=frontend", "--local"},
		},
		{
			name:     "local conflicts with server dry run",
			args:     []string{"set", "selector", "-f", "service.yaml", "app=frontend", "--local", "--dry-run=server"},
			rejected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if !test.rejected {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t, clitest.InspectionRejected(), clitest.FailsWith(clitest.DiagnosticAt(0).Present()))
		})
	}
}

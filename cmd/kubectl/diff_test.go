package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestDiffDeclaresSourceAndServerSideRelationships(t *testing.T) {
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
			name: "filename source",
			args: []string{"diff", "-f", "object.yaml"},
		},
		{
			name: "kustomize source",
			args: []string{"diff", "-k", "overlay"},
		},
		{
			name:     "source required",
			args:     []string{"diff"},
			rejected: true,
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"diff", "-f", "object.yaml", "-k", "overlay"},
			rejected: true,
		},
		{
			name:     "recursive kustomize conflict",
			args:     []string{"diff", "-k", "overlay", "-R"},
			rejected: true,
		},
		{
			name:     "force conflicts requires server side",
			args:     []string{"diff", "-f", "object.yaml", "--force-conflicts"},
			rejected: true,
		},
		{
			name: "force conflicts with server side",
			args: []string{"diff", "-f", "object.yaml", "--server-side", "--force-conflicts"},
		},
		{
			name: "assigned short filename",
			args: []string{"diff", "-f=object.yaml"},
		},
		{
			name: "repeated field manager keeps last",
			args: []string{"diff", "-f", "object.yaml", "--field-manager=first", "--field-manager=second"},
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

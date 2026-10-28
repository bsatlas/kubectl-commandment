package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestApplyDeclaresSourceAndPolicyRelationships(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name     string
		args     []string
		env      cli.Env
		rejected bool
	}{
		{
			name: "filename source",
			args: []string{"apply", "-f", "object.yaml"},
		},
		{
			name: "kustomize source",
			args: []string{"apply", "-k", "overlay"},
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"apply", "-f", "object.yaml", "-k", "overlay"},
			rejected: true,
		},
		{
			name:     "force conflicts requires server side",
			args:     []string{"apply", "-f", "object.yaml", "--force-conflicts"},
			rejected: true,
		},
		{
			name:     "prune requires selection",
			args:     []string{"apply", "-f", "object.yaml", "--prune"},
			rejected: true,
		},
		{
			name: "prune with all",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--all"},
		},
		{
			name:     "applyset unavailable without feature environment",
			args:     []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example"},
			rejected: true,
		},
		{
			name: "applyset enabled by environment",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "TrUe",
				},
			},
		},
		{
			name: "applyset permits server-side prune",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example", "--server-side"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
		},
		{
			name: "applyset still requires prune",
			args: []string{"apply", "-f", "object.yaml", "--applyset=example"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "true",
				},
			},
			rejected: true,
		},
		{
			name: "non-true applyset environment leaves flag unavailable",
			args: []string{"apply", "-f", "object.yaml", "--prune", "--applyset=example"},
			env: cli.Env{
				{
					Name:  "KUBECTL_APPLYSET",
					Value: "1",
				},
			},
			rejected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := cli.InspectionRequest{
				Args: test.args,
				Env:  test.env,
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

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetSubjectDeclaresSourceSubjectAndServiceAccountSemantics(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "resource and user",
			args: []string{"set", "subject", "rolebinding/admin", "--user=alice"},
		},
		{
			name: "filename and group",
			args: []string{"set", "subject", "-f", "binding.yaml", "--group=dev"},
		},
		{
			name:     "subject required",
			args:     []string{"set", "subject", "rolebinding/admin"},
			rejected: true,
		},
		{
			name:     "source required",
			args:     []string{"set", "subject", "--user=alice"},
			rejected: true,
		},
		{
			name:     "service account requires namespace separator",
			args:     []string{"set", "subject", "rolebinding/admin", "--serviceaccount=builder"},
			rejected: true,
		},
		{
			name:     "service account requires name",
			args:     []string{"set", "subject", "rolebinding/admin", "--serviceaccount=default:"},
			rejected: true,
		},
		{
			name: "role binding permits default namespace",
			args: []string{"set", "subject", "rolebinding/admin", "--serviceaccount=:builder"},
		},
		{
			name:     "local forbids resource arguments",
			args:     []string{"set", "subject", "rolebinding/admin", "--user=alice", "--local"},
			rejected: true,
		},
		{
			name:     "all conflicts with selector",
			args:     []string{"set", "subject", "rolebinding", "--user=alice", "--all", "--selector=team=dev"},
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

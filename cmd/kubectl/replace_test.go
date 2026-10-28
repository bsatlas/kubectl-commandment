package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestReplaceDeclaresSourceForceAndRawGrammar(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "filename",
			args:  []string{"replace", "-f", "resource.yaml"},
			ready: true,
		},
		{
			name:  "kustomize",
			args:  []string{"replace", "-k", "dir"},
			ready: true,
		},
		{
			name:  "missing source",
			args:  []string{"replace"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"replace", "-f", "a.yaml", "-k", "dir"},
			ready: false,
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"replace", "-f", "a.yaml", "--kustomize="},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"replace", "--kustomize="},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"replace", "-k", "dir", "--recursive"},
			ready: false,
		},
		{
			name:  "grace without force",
			args:  []string{"replace", "-f", "a.yaml", "--grace-period=0"},
			ready: false,
		},
		{
			name:  "timeout without force",
			args:  []string{"replace", "-f", "a.yaml", "--timeout=1s"},
			ready: false,
		},
		{
			name:  "force server dry run",
			args:  []string{"replace", "-f", "a.yaml", "--force", "--dry-run=server"},
			ready: false,
		},
		{
			name:  "grace with force",
			args:  []string{"replace", "-f", "a.yaml", "--grace-period=0", "--force"},
			ready: true,
		},
		{
			name:  "timeout with force",
			args:  []string{"replace", "-f", "a.yaml", "--timeout=1s", "--force"},
			ready: true,
		},
		{
			name:  "raw filename",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml"},
			ready: true,
		},
		{
			name:  "raw invalid URI",
			args:  []string{"replace", "--raw=://bad", "-f", "a.yaml"},
			ready: false,
		},
		{
			name:  "raw remote filename",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "https://example.invalid/a.yaml"},
			ready: false,
		},
		{
			name:  "raw output",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml", "--output=name"},
			ready: false,
		},
		{
			name:  "raw multiple files",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml,b.yaml"},
			ready: false,
		},
		{
			name:  "raw recursive",
			args:  []string{"replace", "--raw=/api/v1/pods/foo", "-f", "a.yaml", "--recursive"},
			ready: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			inspection := clitest.Inspect(t.Context(), app, request)
			if test.ready {
				inspection.Require(t, clitest.InspectionReady())
				return
			}
			inspection.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

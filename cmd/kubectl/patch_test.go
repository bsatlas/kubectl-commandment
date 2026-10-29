package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestPatchDeclaresPatchAndResourceGrammar(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "server resource patch",
			args:  []string{"patch", "pod/foo", "--patch={}"},
			ready: true,
		},
		{
			name:  "file resource patch",
			args:  []string{"patch", "-f", "pod.yaml", "--patch={}"},
			ready: true,
		},
		{
			name:  "case insensitive patch type",
			args:  []string{"patch", "pod/foo", "--patch={}", "--type=MERGE"},
			ready: true,
		},
		{
			name:  "explicit empty patch type uses strategic default",
			args:  []string{"patch", "pod/foo", "--patch={}", "--type="},
			ready: true,
		},
		{
			name:  "kustomize resource patch",
			args:  []string{"patch", "-k", "dir", "--patch={}"},
			ready: true,
		},
		{
			name:  "invalid patch type",
			args:  []string{"patch", "pod/foo", "--patch={}", "--type=unknown"},
			ready: false,
		},
		{
			name:  "missing patch",
			args:  []string{"patch", "pod/foo"},
			ready: false,
		},
		{
			name:  "empty inline patch",
			args:  []string{"patch", "pod/foo", "--patch="},
			ready: false,
		},
		{
			name:  "empty inline patch does not conflict with patch file",
			args:  []string{"patch", "pod/foo", "--patch=", "--patch-file=patch.yaml"},
			ready: true,
		},
		{
			name:  "both patch sources",
			args:  []string{"patch", "pod/foo", "--patch={}", "--patch-file=patch.yaml"},
			ready: false,
		},
		{
			name:  "missing resource source",
			args:  []string{"patch", "--patch={}"},
			ready: false,
		},
		{
			name:  "local server resource",
			args:  []string{"patch", "pod/foo", "--patch={}", "--local"},
			ready: false,
		},
		{
			name:  "local server dry run",
			args:  []string{"patch", "-f", "pod.yaml", "--patch={}", "--local", "--dry-run=server"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"patch", "-f", "a.yaml", "-k", "dir", "--patch={}"},
			ready: false,
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"patch", "-f", "a.yaml", "--kustomize=", "--patch={}"},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"patch", "--kustomize=", "--patch={}"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"patch", "-k", "dir", "--recursive", "--patch={}"},
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

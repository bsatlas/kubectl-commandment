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
			name:  "invalid dry run",
			args:  []string{"patch", "pod/foo", "--patch={}", "--dry-run=wat"},
			ready: false,
		},
		{
			name:  "invalid output format",
			args:  []string{"patch", "pod/foo", "--patch={}", "--output=wat"},
			ready: false,
		},
		{
			name:  "server resource patch",
			args:  []string{"patch", "pod/foo", "--patch={}"},
			ready: true,
		},
		{
			name:  "assigned short filename and patch",
			args:  []string{"patch", "-f=pod.yaml", "-p={}"},
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

func TestPatchRepeatedScalarUsesLastOccurrence(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"patch", "pod/foo", "--patch={}", "--field-manager=first", "--field-manager=second"},
	}
	inspection := clitest.Inspect(t.Context(), app, request)
	inspection.Require(t,
		clitest.InspectionReady(),
		clitest.RootCommand().Command("patch").Flag("--field-manager").Value().Text("second"),
	)
}

package kubectl_test

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
	"github.com/abstractionsinc/kubectl-commandment/internal/kubectl"
)

func TestAnnotateDeclaresResourceAndUpdateGrammar(t *testing.T) {
	t.Parallel()

	app, err := kubectl.Application(kubectl.PluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name  string
		args  []string
		ready bool
	}{
		{
			name:  "resource update",
			args:  []string{"annotate", "pods", "foo", "description=frontend"},
			ready: true,
		},
		{
			name:  "list without update",
			args:  []string{"annotate", "pods", "foo", "--list"},
			ready: true,
		},
		{
			name:  "missing update",
			args:  []string{"annotate", "pods", "foo"},
			ready: false,
		},
		{
			name:  "list with update",
			args:  []string{"annotate", "pods", "foo", "description=frontend", "--list"},
			ready: false,
		},
		{
			name:  "all with selector",
			args:  []string{"annotate", "pods", "description=frontend", "--all", "--selector=app=web"},
			ready: false,
		},
		{
			name:  "list with output",
			args:  []string{"annotate", "pods", "foo", "--list", "--output=name"},
			ready: false,
		},
		{
			name:  "all with field selector",
			args:  []string{"annotate", "pods", "description=frontend", "--all", "--field-selector=metadata.name=foo"},
			ready: false,
		},
		{
			name:  "missing resource source",
			args:  []string{"annotate", "description=frontend"},
			ready: false,
		},
		{
			name:  "modify and remove same annotation",
			args:  []string{"annotate", "pods", "foo", "description=frontend", "description-"},
			ready: false,
		},
		{
			name:  "local file update",
			args:  []string{"annotate", "-f", "pod.yaml", "description=frontend", "--local"},
			ready: true,
		},
		{
			name:  "kustomize follows upstream recursive default",
			args:  []string{"annotate", "-k", "dir", "description=frontend"},
			ready: false,
		},
		{
			name:  "filename and kustomize",
			args:  []string{"annotate", "-f", "pod.yaml", "-k", "dir", "description=frontend"},
			ready: false,
		},
		{
			name:  "filename and explicit empty kustomize",
			args:  []string{"annotate", "-f", "pod.yaml", "--kustomize=", "description=frontend"},
			ready: true,
		},
		{
			name:  "explicit empty kustomize is not a source",
			args:  []string{"annotate", "--kustomize=", "description=frontend"},
			ready: false,
		},
		{
			name:  "recursive kustomize",
			args:  []string{"annotate", "-k", "dir", "--recursive", "description=frontend"},
			ready: false,
		},
		{
			name:  "local server dry run",
			args:  []string{"annotate", "-f", "pod.yaml", "description=frontend", "--local", "--dry-run=server"},
			ready: false,
		},
		{
			name:  "local without file",
			args:  []string{"annotate", "description=frontend", "--local"},
			ready: false,
		},
		{
			name:  "local resource argument",
			args:  []string{"annotate", "pods", "foo", "description=frontend", "--local", "-f", "pod.yaml"},
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

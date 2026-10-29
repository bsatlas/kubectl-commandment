package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestSetImageDeclaresSourceAndWildcardRelationships(t *testing.T) {
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
			name: "resource and image",
			args: []string{"set", "image", "deployment/nginx", "web=nginx:latest"},
		},
		{
			name: "image value ending in dash",
			args: []string{"set", "image", "deployment/nginx", "web=example-"},
		},
		{
			name: "filename and image",
			args: []string{"set", "image", "-f", "object.yaml", "web=nginx:latest"},
		},
		{
			name: "multiple image updates",
			args: []string{"set", "image", "deployment/nginx", "web=nginx:latest", "sidecar=busybox:latest"},
		},
		{
			name:     "resource after image update is rejected",
			args:     []string{"set", "image", "deployment/nginx", "web=nginx:latest", "service/backend"},
			rejected: true,
		},
		{
			name:     "source required",
			args:     []string{"set", "image", "web=nginx:latest"},
			rejected: true,
		},
		{
			name: "wildcard image alone",
			args: []string{"set", "image", "deployment/nginx", "*=nginx:latest"},
		},
		{
			name:     "wildcard image is exclusive",
			args:     []string{"set", "image", "deployment/nginx", "*=nginx:latest", "sidecar=busybox"},
			rejected: true,
		},
		{
			name:     "image update required",
			args:     []string{"set", "image", "deployment/nginx"},
			rejected: true,
		},
		{
			name:     "filename and kustomize conflict",
			args:     []string{"set", "image", "-f", "object.yaml", "-k", "overlay", "web=nginx:latest"},
			rejected: true,
		},
		{
			name:     "recursive kustomize conflict",
			args:     []string{"set", "image", "deployment/nginx", "web=nginx:latest", "-k", "overlay", "-R"},
			rejected: true,
		},
		{
			name:     "all conflicts with selector",
			args:     []string{"set", "image", "deployment", "web=nginx:latest", "--all", "--selector=app=web"},
			rejected: true,
		},
		{
			name:     "local forbids resource arguments",
			args:     []string{"set", "image", "deployment/nginx", "web=nginx:latest", "--local"},
			rejected: true,
		},
		{
			name: "local filename source",
			args: []string{"set", "image", "-f", "object.yaml", "web=nginx:latest", "--local"},
		},
		{
			name:     "local conflicts with server dry run",
			args:     []string{"set", "image", "-f", "object.yaml", "web=nginx:latest", "--local", "--dry-run=server"},
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

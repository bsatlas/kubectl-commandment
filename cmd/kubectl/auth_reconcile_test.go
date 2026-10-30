package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestAuthReconcileSourceAndDryRunPolicy(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	tests := []struct {
		name     string
		args     []string
		rejected bool
	}{
		{
			name: "filename",
			args: []string{"auth", "reconcile", "-f", "rbac.yaml"},
		},
		{
			name: "joined short filename",
			args: []string{"auth", "reconcile", "-frbac.yaml"},
		},
		{
			name: "assigned short filename",
			args: []string{"auth", "reconcile", "-f=rbac.yaml"},
		},
		{
			name: "kustomize",
			args: []string{"auth", "reconcile", "-k", "overlay"},
		},
		{
			name:     "missing source",
			args:     []string{"auth", "reconcile"},
			rejected: true,
		},
		{
			name:     "filename and kustomize",
			args:     []string{"auth", "reconcile", "-f", "rbac.yaml", "-k", "overlay"},
			rejected: true,
		},
		{
			name:     "recursive kustomize",
			args:     []string{"auth", "reconcile", "-k", "overlay", "-R"},
			rejected: true,
		},
		{
			name: "client dry run",
			args: []string{"auth", "reconcile", "-f", "rbac.yaml", "--dry-run=client"},
		},
		{
			name:     "server dry run",
			args:     []string{"auth", "reconcile", "-f", "rbac.yaml", "--dry-run=server"},
			rejected: true,
		},
		{
			name:     "positional argument",
			args:     []string{"auth", "reconcile", "-f", "rbac.yaml", "extra"},
			rejected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			proof := clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: test.args})
			if !test.rejected {
				proof.Require(t, clitest.InspectionReady())
				return
			}
			proof.Require(t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestAuthReconcileUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}
	tests := []struct {
		name string
		args []string
		flag string
		want string
	}{
		{
			name: "dry run",
			args: []string{"auth", "reconcile", "-f", "rbac.yaml", "--dry-run=server", "--dry-run=client"},
			flag: "--dry-run",
			want: "client",
		},
		{
			name: "kustomize",
			args: []string{"auth", "reconcile", "-k", "first", "-k", "second"},
			flag: "--kustomize",
			want: "second",
		},
		{
			name: "recursive",
			args: []string{"auth", "reconcile", "-f", "rbac.yaml", "--recursive", "--recursive=false"},
			flag: "--recursive",
			want: "false",
		},
		{
			name: "remove extra permissions",
			args: []string{"auth", "reconcile", "-f", "rbac.yaml", "--remove-extra-permissions", "--remove-extra-permissions=false"},
			flag: "--remove-extra-permissions",
			want: "false",
		},
		{
			name: "remove extra subjects",
			args: []string{"auth", "reconcile", "-f", "rbac.yaml", "--remove-extra-subjects", "--remove-extra-subjects=false"},
			flag: "--remove-extra-subjects",
			want: "false",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value := clitest.Command("auth", "reconcile").Flag(test.flag).Value()
			clitest.Inspect(t.Context(), app, cli.InspectionRequest{Args: test.args}).Require(t,
				clitest.InspectionReady(),
				value.Resolved(),
				value.Text(test.want),
			)
		})
	}
}

func TestAuthReconcileFilenameUsesStringSliceCSVLanguage(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("application() error = %v", err)
	}

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"auth", "reconcile", "--filename", `one.yaml,"two,three.yaml"`},
	}).Require(t, clitest.InspectionReady())

	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"auth", "reconcile", "--filename", `"unterminated`},
	}).Require(t,
		clitest.InspectionRejected(),
		clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
	)
}

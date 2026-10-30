package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateRoleBindingFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "rolebinding role reference is exclusive",
			args: []string{"create", "rolebinding", "reader", "--role=reader", "--clusterrole=reader"},
		},
		{
			name: "rolebinding requires role reference",
			args: []string{"create", "rolebinding", "reader"},
		},
		{
			name: "rolebinding serviceaccount has kubectl syntax",
			args: []string{"create", "rolebinding", "reader", "--role=reader", "--serviceaccount=ns:sa:extra"},
		},
		{
			name: "rolebinding validates nonempty name",
			args: []string{"create", "rolebinding", "", "--role=reader"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(
				t,
				clitest.InspectionRejected(),
				clitest.FailsWith(clitest.DiagnosticAt(0).Present()),
			)
		})
	}
}

func TestCreateRoleBindingFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "rolebinding", "reader", "--role=reader", "--field-manager=first", "--field-manager=second"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "rolebinding").Flag("--field-manager").Value().Text("second"),
	)
}

package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestCreateClusterRoleFrontendRejectsInvalidKubectlSemantics(t *testing.T) {
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
			name: "clusterrole requires verbs",
			args: []string{"create", "clusterrole", "reader", "--resource=pods"},
		},
		{
			name: "clusterrole requires a target",
			args: []string{"create", "clusterrole", "reader", "--verb=get"},
		},
		{
			name: "clusterrole aggregation excludes rules",
			args: []string{"create", "clusterrole", "reader", "--aggregation-rule=app=reader", "--verb=get", "--resource=pods"},
		},
		{
			name: "clusterrole non resource verbs are bounded",
			args: []string{"create", "clusterrole", "reader", "--verb=watch", "--non-resource-url=/healthz"},
		},
		{
			name: "clusterrole non resource url has kubectl syntax",
			args: []string{"create", "clusterrole", "reader", "--verb=get", "--non-resource-url=healthz"},
		},
		{
			name: "clusterrole non resource wildcard is suffix only",
			args: []string{"create", "clusterrole", "reader", "--verb=get", "--non-resource-url=/health*/ready"},
		},
		{
			name: "clusterrole resource has kubectl syntax",
			args: []string{"create", "clusterrole", "reader", "--verb=get", "--resource=.apps"},
		},
		{
			name: "clusterrole validates nonempty name",
			args: []string{"create", "clusterrole", "", "--verb=get", "--resource=pods"},
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

func TestCreateClusterRoleFrontendAcceptsKubectlBoundaryCases(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	tests := []struct {
		name         string
		args         []string
		expectations []clitest.InspectionExpectation
	}{
		{
			name: "clusterrole aggregation stands alone",
			args: []string{"create", "clusterrole", "aggregate", "--aggregation-rule=app=reader"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			expectations := []clitest.InspectionExpectation{clitest.InspectionReady()}
			expectations = append(expectations, test.expectations...)
			request := cli.InspectionRequest{
				Args: test.args,
			}
			clitest.Inspect(t.Context(), app, request).Require(t, expectations...)
		})
	}
}

func TestCreateClusterRoleFrontendUsesPflagLastOccurrenceSemantics(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{
		Args: []string{"create", "clusterrole", "reader", "--verb=get", "--resource=pods", "--field-manager=first", "--field-manager=second"},
	}
	clitest.Inspect(t.Context(), app, request).Require(
		t,
		clitest.InspectionReady(),
		clitest.Command("create", "clusterrole").Flag("--field-manager").Value().Text("second"),
	)
}

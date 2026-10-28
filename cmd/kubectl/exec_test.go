package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestExecFrontendContract(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{
		Args: []string{"exec", "pod-a", "--", "sh", "--container=remote"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(1).Role(cli.TokenRolePositional),
		clitest.TokenAt(2).Role(cli.TokenRoleEndOfOptions),
		clitest.TokenAt(3).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(4).Role(cli.TokenRolePassThrough),
	)

	withoutSeparator := cli.InspectionRequest{
		Args: []string{"exec", "pod-a", "sh"},
	}
	clitest.Inspect(t.Context(), app, withoutSeparator).Require(t,
		clitest.InspectionRejected(),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"exec", "--filename=pod.yaml", "--", "sh"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"exec", "--", "sh"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"exec", "pod-a", "--"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"exec", "pod-a", "--pod-running-timeout=0s", "--", "sh"},
	}).Require(t, clitest.InspectionRejected())
}

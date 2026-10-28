package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestRunFrontendContract(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--", "arg", "--image=remote"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(1).Role(cli.TokenRolePositional),
		clitest.TokenAt(3).Role(cli.TokenRoleEndOfOptions),
		clitest.TokenAt(4).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(5).Role(cli.TokenRolePassThrough),
	)

	nameAfterSeparator := cli.InspectionRequest{
		Args: []string{"run", "--", "work"},
	}
	clitest.Inspect(t.Context(), app, nameAfterSeparator).Require(t,
		clitest.InspectionRejected(),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--tty"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--expose"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--rm"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--attach", "--dry-run=client"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--dry-run"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--dry-run=true"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--dry-run=false"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--dry-run=bogus"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--env=TOKEN=a=b", "--annotations=owner=a=b"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"run", "work", "--image=busybox", "--port=not-a-number"},
	}).Require(t, clitest.InspectionRejected())
}

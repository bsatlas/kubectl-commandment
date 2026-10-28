package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestDebugFrontendContract(t *testing.T) {
	t.Parallel()

	app := clitest.Compile(t, applicationDefinition(pluginCatalog{}))
	request := cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "pod-b", "--", "sh", "--image=busybox"},
	}
	clitest.Inspect(t.Context(), app, request).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(1).Role(cli.TokenRolePositional),
		clitest.TokenAt(2).Role(cli.TokenRolePositional),
		clitest.TokenAt(3).Role(cli.TokenRoleEndOfOptions),
		clitest.TokenAt(4).Role(cli.TokenRolePassThrough),
		clitest.TokenAt(5).Role(cli.TokenRolePassThrough),
	)

	withoutSeparator := cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "pod-b", "--image=busybox"},
	}
	clitest.Inspect(t.Context(), app, withoutSeparator).Require(t,
		clitest.InspectionReady(),
		clitest.TokenAt(1).Role(cli.TokenRolePositional),
		clitest.TokenAt(2).Role(cli.TokenRolePositional),
	)
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "--copy-to=pod-copy"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "--copy-to=pod-copy", "--set-image=*=busybox"},
	}).Require(t, clitest.InspectionReady())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "--copy-to=pod-copy", "--target=main", "--image=busybox"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "--copy-to=pod-copy", "--set-image=*=busybox", "--attach"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "--copy-to=pod-copy", "--", "echo"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "--image=busybox", "--replace"},
	}).Require(t, clitest.InspectionRejected())
	clitest.Inspect(t.Context(), app, cli.InspectionRequest{
		Args: []string{"debug", "pod-a", "--image=busybox", "--tty"},
	}).Require(t, clitest.InspectionRejected())
}

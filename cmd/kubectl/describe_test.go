package main

import (
	"testing"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/abstractionsinc/commandment/clitest"
)

func TestDescribeAcceptsNamedResourceWithSelector(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"describe", "pods", "pod-a", "-l=app=test"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

func TestDescribeAllowsBulkResourceWithSelector(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"describe", "pods", "--selector", "app=test"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

func TestDescribeAcceptsResourceArgumentsWithFilename(t *testing.T) {
	t.Parallel()

	app, err := application(pluginCatalog{})
	if err != nil {
		t.Fatalf("Application() error = %v", err)
	}
	request := cli.InspectionRequest{Args: []string{"describe", "pods", "--filename", "pod.yaml"}}

	clitest.Inspect(t.Context(), app, request).Require(t, clitest.InspectionReady())
}

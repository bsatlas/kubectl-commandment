package main

import (
	"context"
	"fmt"

	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func runAPIVersions(_ context.Context, cfg *apiVersionsConfig) error {
	discoveryClient, err := restClientGetter(cfg.Client).ToDiscoveryClient()
	if err != nil {
		return err
	}
	discoveryClient.Invalidate()

	groups, err := discoveryClient.ServerGroups()
	if err != nil {
		return fmt.Errorf("couldn't get available api versions from server: %v", err)
	}
	versions := v1.ExtractGroupVersions(groups)
	slices.Sort(versions)
	for _, version := range versions {
		_, _ = fmt.Fprintln(cfg.Stdout, version)
	}
	return nil
}

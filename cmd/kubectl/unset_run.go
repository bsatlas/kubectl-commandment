package main

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

func runConfigUnset(_ context.Context, cfg *configUnsetConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	if err := unsetConfigProperty(config, cfg.Property); err != nil {
		return err
	}
	if err := clientcmd.ModifyConfig(access, *config, false); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "Property %q unset.\n", cfg.Property.Path)
	return err
}

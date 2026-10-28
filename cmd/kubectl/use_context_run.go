package main

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

func runUseContext(_ context.Context, cfg *useContextConfig) error {
	access := configAccess(cfg.Kubeconfig)
	starting, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	if _, exists := starting.Contexts[cfg.Context]; !exists {
		return fmt.Errorf("no context exists with the name: %q", cfg.Context)
	}
	starting.CurrentContext = cfg.Context
	if err := clientcmd.ModifyConfig(access, *starting, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "Switched to context %q.\n", cfg.Context)
	return err
}

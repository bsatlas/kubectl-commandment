package main

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

func runConfigRenameContext(_ context.Context, cfg *configRenameContextConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	file := configFile(access)
	currentContext, exists := config.Contexts[cfg.Name]
	if !exists {
		return fmt.Errorf("cannot rename the context %q, it's not in %s", cfg.Name, file)
	}
	if _, exists := config.Contexts[cfg.NewName]; exists {
		return fmt.Errorf("cannot rename the context %q, the context %q already exists in %s", cfg.Name, cfg.NewName, file)
	}
	config.Contexts[cfg.NewName] = currentContext
	delete(config.Contexts, cfg.Name)
	if config.CurrentContext == cfg.Name {
		config.CurrentContext = cfg.NewName
	}
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "Context %q renamed to %q.\n", cfg.Name, cfg.NewName)
	return err
}

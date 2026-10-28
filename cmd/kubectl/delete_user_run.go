package main

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

func runConfigDeleteUser(_ context.Context, cfg *configDeleteUserConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	file := configFile(access)
	if _, ok := config.AuthInfos[cfg.Name]; !ok {
		return fmt.Errorf("cannot delete user %s, not in %s", cfg.Name, file)
	}
	delete(config.AuthInfos, cfg.Name)
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "deleted user %s from %s\n", cfg.Name, file)
	return err
}

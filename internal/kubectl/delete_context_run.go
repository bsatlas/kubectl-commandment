package kubectl

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

func runConfigDeleteContext(_ context.Context, cfg *configDeleteContextConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	file := configFile(access)
	if _, ok := config.Contexts[cfg.Name]; !ok {
		return fmt.Errorf("cannot delete context %s, not in %s", cfg.Name, file)
	}
	if config.CurrentContext == cfg.Name {
		if _, err := fmt.Fprint(cfg.Stderr, "warning: this removed your active context, use \"kubectl config use-context\" to select a different one\n"); err != nil {
			return err
		}
	}
	delete(config.Contexts, cfg.Name)
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "deleted context %s from %s\n", cfg.Name, file)
	return err
}

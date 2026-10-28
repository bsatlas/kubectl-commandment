package kubectl

import (
	"context"
	"fmt"
)

func runConfigGetClusters(_ context.Context, cfg *configGetClustersConfig) error {
	config, err := configAccess(cfg.Kubeconfig).GetStartingConfig()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cfg.Stdout, "NAME"); err != nil {
		return err
	}
	for name := range config.Clusters {
		if _, err := fmt.Fprintln(cfg.Stdout, name); err != nil {
			return err
		}
	}
	return nil
}

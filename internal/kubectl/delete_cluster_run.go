package kubectl

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"
)

func runConfigDeleteCluster(_ context.Context, cfg *configDeleteClusterConfig) error {
	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}
	file := configFile(access)
	if _, ok := config.Clusters[cfg.Name]; !ok {
		return fmt.Errorf("cannot delete cluster %s, not in %s", cfg.Name, file)
	}
	delete(config.Clusters, cfg.Name)
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "deleted cluster %s from %s\n", cfg.Name, file)
	return err
}

package kubectl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func runConfigSetCluster(_ context.Context, cfg *configSetClusterConfig) error {
	if cfg.EmbedCerts {
		if _, err := os.Stat(cfg.CertificateAuthority); err != nil {
			return fmt.Errorf("could not stat %s file %s: %v", clientcmd.FlagCAFile, cfg.CertificateAuthority, err)
		}
	}

	access := configAccess(cfg.Kubeconfig)
	config, err := access.GetStartingConfig()
	if err != nil {
		return err
	}

	cluster, exists := config.Clusters[cfg.Name]
	if !exists {
		cluster = api.NewCluster()
	}
	modified := *cluster

	if flagProvided(cfg.ServerState) {
		modified.Server = cfg.Server
		modified.TLSServerName = ""
	}
	if flagProvided(cfg.TLSServerNameState) {
		modified.TLSServerName = cfg.TLSServerName
	}
	if flagProvided(cfg.InsecureState) {
		modified.InsecureSkipTLSVerify = cfg.InsecureSkipTLSVerify
		if modified.InsecureSkipTLSVerify {
			modified.CertificateAuthority = ""
			modified.CertificateAuthorityData = nil
		}
	}
	if flagProvided(cfg.CertificateAuthorityState) {
		if cfg.EmbedCerts {
			modified.CertificateAuthorityData, err = os.ReadFile(cfg.CertificateAuthority)
			if err != nil {
				return err
			}
			modified.InsecureSkipTLSVerify = false
			modified.CertificateAuthority = ""
		} else {
			path, pathErr := filepath.Abs(cfg.CertificateAuthority)
			if pathErr != nil {
				return pathErr
			}
			modified.CertificateAuthority = path
			if path != "" {
				modified.InsecureSkipTLSVerify = false
				modified.CertificateAuthorityData = nil
			}
		}
	}
	if flagProvided(cfg.ProxyURLState) {
		modified.ProxyURL = cfg.ProxyURL
	}

	config.Clusters[cfg.Name] = &modified
	if err := clientcmd.ModifyConfig(access, *config, true); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.Stdout, "Cluster %q set.\n", cfg.Name)
	return err
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type configSetClusterConfig struct {
	Kubeconfig                string
	Name                      string
	CertificateAuthority      string
	EmbedCerts                bool
	InsecureSkipTLSVerify     bool
	ProxyURL                  string
	Server                    string
	TLSServerName             string
	CertificateAuthorityState cli.OptionalValue[cli.ValueState]
	InsecureState             cli.OptionalValue[cli.ValueState]
	ProxyURLState             cli.OptionalValue[cli.ValueState]
	ServerState               cli.OptionalValue[cli.ValueState]
	TLSServerNameState        cli.OptionalValue[cli.ValueState]
	Stdout                    io.Writer
}

func configSetClusterCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configSetClusterConfig] {
	cmd := cli.Command[configSetClusterConfig]{
		Name: "set-cluster",
		Description: cli.Description{
			Short: "Set a cluster entry in kubeconfig",
			Long:  "Set a cluster entry in kubeconfig.\n\n Specifying a name that already exists will merge new fields on top of existing values for those fields.",
		},
		Examples: []cli.Example{
			{
				Description: "Set only the server field on the e2e cluster entry without touching other values",
				CommandLine: "kubectl config set-cluster e2e --server=https://1.2.3.4",
			},
			{
				Description: "Embed certificate authority data for the e2e cluster entry",
				CommandLine: "kubectl config set-cluster e2e --embed-certs --certificate-authority=~/.kube/e2e/kubernetes.ca.crt",
			},
			{
				Description: "Disable cert checking for the e2e cluster entry",
				CommandLine: "kubectl config set-cluster e2e --insecure-skip-tls-verify=true",
			},
			{
				Description: "Set the custom TLS server name to use for validation for the e2e cluster entry",
				CommandLine: "kubectl config set-cluster e2e --tls-server-name=my-cluster-name",
			},
			{
				Description: "Set the proxy URL for the e2e cluster entry",
				CommandLine: "kubectl config set-cluster e2e --proxy-url=https://1.2.3.4",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configSetClusterConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			name := cb.Values.String(&cb.Config.Name)
			cb.Arguments.ValueFor(name, "NAME")

			certificateAuthority := cb.Values.String(&cb.Config.CertificateAuthority)
			cb.Flags.ValueFor(certificateAuthority).Description("Path to certificate-authority file for the cluster entry in kubeconfig").Long("certificate-authority")
			cb.Access.ValueState(certificateAuthority, &cb.Config.CertificateAuthorityState)

			embedCerts := cb.Values.Bool(&cb.Config.EmbedCerts)
			cb.Flags.ValueFor(embedCerts).Description("embed-certs for the cluster entry in kubeconfig").Long("embed-certs").ImplicitValue(true)

			insecure := cb.Values.Bool(&cb.Config.InsecureSkipTLSVerify)
			cb.Values.Default(insecure, false)
			cb.Flags.ValueFor(insecure).Description("insecure-skip-tls-verify for the cluster entry in kubeconfig").Long("insecure-skip-tls-verify").ImplicitValue(true)
			cb.Access.ValueState(insecure, &cb.Config.InsecureState)

			proxyURL := cb.Values.String(&cb.Config.ProxyURL)
			cb.Flags.ValueFor(proxyURL).Description("proxy-url for the cluster entry in kubeconfig").Long("proxy-url")
			cb.Access.ValueState(proxyURL, &cb.Config.ProxyURLState)

			server := cb.Values.String(&cb.Config.Server)
			cb.Flags.ValueFor(server).Description("server for the cluster entry in kubeconfig").Long("server")
			cb.Access.ValueState(server, &cb.Config.ServerState)

			tlsServerName := cb.Values.String(&cb.Config.TLSServerName)
			cb.Flags.ValueFor(tlsServerName).Description("tls-server-name for the cluster entry in kubeconfig").Long("tls-server-name")
			cb.Access.ValueState(tlsServerName, &cb.Config.TLSServerNameState)
		},
		Run: runConfigSetCluster,
	}
	return cmd
}

func runConfigSetCluster(_ context.Context, cfg *configSetClusterConfig) error {
	if cfg.Name == "" {
		return errors.New("you must specify a non-empty cluster name")
	}
	if cfg.InsecureSkipTLSVerify && cfg.CertificateAuthority != "" {
		return errors.New("you cannot specify a certificate authority and insecure mode at the same time")
	}
	if cfg.EmbedCerts {
		if cfg.CertificateAuthority == "" {
			return fmt.Errorf("you must specify a --%s to embed", clientcmd.FlagCAFile)
		}
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

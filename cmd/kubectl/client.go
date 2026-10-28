package main

import (
	"context"
	"errors"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type clientConfig struct {
	CacheDir             string
	Kubeconfig           string
	Cluster              string
	User                 string
	Context              string
	Namespace            string
	Server               string
	TLSServerName        string
	Insecure             bool
	ClientCertificate    string
	ClientKey            string
	CertificateAuthority string
	Token                string
	Impersonate          string
	ImpersonateUID       string
	ImpersonateGroups    []string
	ImpersonateUserExtra []string
	Username             string
	Password             string
	RequestTimeout       string
	DisableCompression   bool
	ProxyURL             string
}

func deriveClientConfig[C any](command *cli.CommandBuilder[C], storage *clientConfig, source cli.Destination[rootConfig, clientConfig]) cli.Destination[C, clientConfig] {
	return command.Values.Derive(storage, cli.Value.ComputedBy(func(_ context.Context, _ *C, dependencies cli.Dependencies) (clientConfig, error) {
		value, ok := dependencies.Get(source)
		if !ok {
			return clientConfig{}, errors.New("client configuration dependency is unavailable")
		}
		return value, nil
	}).From(source))
}

func restClientGetter(selection clientConfig) genericclioptions.RESTClientGetter {
	flags := genericclioptions.NewConfigFlags(true).
		WithDeprecatedPasswordFlag().
		WithDiscoveryBurst(300).
		WithDiscoveryQPS(50.0)
	*flags.CacheDir = selection.CacheDir
	*flags.KubeConfig = selection.Kubeconfig
	*flags.ClusterName = selection.Cluster
	*flags.AuthInfoName = selection.User
	*flags.Context = selection.Context
	*flags.Namespace = selection.Namespace
	*flags.APIServer = selection.Server
	*flags.TLSServerName = selection.TLSServerName
	*flags.Insecure = selection.Insecure
	*flags.CertFile = selection.ClientCertificate
	*flags.KeyFile = selection.ClientKey
	*flags.CAFile = selection.CertificateAuthority
	*flags.BearerToken = selection.Token
	*flags.Impersonate = selection.Impersonate
	*flags.ImpersonateUID = selection.ImpersonateUID
	*flags.ImpersonateGroup = append((*flags.ImpersonateGroup)[:0], selection.ImpersonateGroups...)
	*flags.ImpersonateUserExtra = append((*flags.ImpersonateUserExtra)[:0], selection.ImpersonateUserExtra...)
	*flags.Username = selection.Username
	*flags.Password = selection.Password
	*flags.Timeout = selection.RequestTimeout
	*flags.DisableCompression = selection.DisableCompression
	*flags.ProxyURL = selection.ProxyURL
	return flags
}

func (config clientConfig) factory() util.Factory {
	return util.NewFactory(restClientGetter(config))
}

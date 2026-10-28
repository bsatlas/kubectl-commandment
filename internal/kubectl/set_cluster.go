package kubectl

import (
	"context"
	"io"

	"github.com/abstractionsinc/commandment/cli"
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
			cb.Rule.NotEmpty(name)

			certificateAuthority := cb.Values.String(&cb.Config.CertificateAuthority)
			cb.Flags.ValueFor(certificateAuthority).
				Description("Path to certificate-authority file for the cluster entry in kubeconfig").
				CompleteWith(cli.CompletionNativeFiles).
				Long("certificate-authority")
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

			cb.Rule.When(cb.When.Equal(insecure, true)).Equal(certificateAuthority, "")
			cb.Rule.Check("kubectl-config-set-cluster-embed-certs", func(_ context.Context, cfg *configSetClusterConfig, _ cli.Dependencies) (bool, error) {
				return !cfg.EmbedCerts || cfg.CertificateAuthority != "", nil
			}).
				DependsOn(embedCerts.OptionalDependency()).
				DependsOn(certificateAuthority.OptionalDependency())
		},
		Run: runConfigSetCluster,
	}
	return cmd
}

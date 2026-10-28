package kubectl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
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
	return newClientConfigFlags(selection)
}

func restClientGetterForInvocation(ctx context.Context, selection clientConfig) genericclioptions.RESTClientGetter {
	flags := newClientConfigFlags(selection)
	return flags.WithWrapConfigFn(func(config *rest.Config) *rest.Config {
		config.Wrap(func(next http.RoundTripper) http.RoundTripper {
			return &invocationRoundTripper{
				ctx:  ctx,
				next: next,
			}
		})
		return config
	})
}

func newClientConfigFlags(selection clientConfig) *genericclioptions.ConfigFlags {
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

func (config clientConfig) factoryForInvocation(ctx context.Context) util.Factory {
	return util.NewFactory(restClientGetterForInvocation(ctx, config))
}

type invocationRoundTripper struct {
	ctx  context.Context
	next http.RoundTripper
}

func (transport *invocationRoundTripper) WrappedRoundTripper() http.RoundTripper {
	return transport.next
}

func (transport *invocationRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if cause := context.Cause(transport.ctx); cause != nil {
		return nil, cause
	}
	requestContext, cancel := context.WithCancelCause(request.Context())
	stop := context.AfterFunc(transport.ctx, func() {
		cancel(context.Cause(transport.ctx))
	})

	response, err := transport.next.RoundTrip(request.WithContext(requestContext))
	if err != nil {
		stop()
		cancel(nil)
		return nil, err
	}

	finish := sync.OnceFunc(func() {
		stop()
		cancel(nil)
	})
	if response.Body == nil {
		finish()
		return response, nil
	}
	response.Body = &invocationResponseBody{
		ReadCloser: response.Body,
		finish:     finish,
	}
	return response, nil
}

type invocationResponseBody struct {
	io.ReadCloser
	finish func()
}

func (body *invocationResponseBody) Read(buffer []byte) (int, error) {
	count, err := body.ReadCloser.Read(buffer)
	if err != nil {
		body.finish()
	}
	return count, err
}

func (body *invocationResponseBody) Close() error {
	err := body.ReadCloser.Close()
	body.finish()
	return err
}

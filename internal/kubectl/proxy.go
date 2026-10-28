package kubectl

import (
	"regexp"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/proxy"
)

type proxyRegexpSet struct {
	Source      string
	Expressions []*regexp.Regexp
}

var proxyRegexpSetLanguage = cli.ValueCodec[proxyRegexpSet]{
	Name: "regular-expression-list",
	Parse: func(value string) (proxyRegexpSet, error) {
		expressions, err := proxy.MakeRegexpArray(value)
		if err != nil {
			return proxyRegexpSet{}, cli.Rejection.Message(
				"regular-expression-list",
				err.Error(),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		set := proxyRegexpSet{
			Source:      value,
			Expressions: expressions,
		}
		return set, nil
	},
	Format: func(value proxyRegexpSet) string { return value.Source },
	Snapshot: func(value proxyRegexpSet) proxyRegexpSet {
		snapshot := proxyRegexpSet{
			Source:      value.Source,
			Expressions: append([]*regexp.Regexp(nil), value.Expressions...),
		}
		return snapshot
	},
}

var proxyPathPrefixLanguage = cli.ValueCodec[string]{
	Name: "proxy-path-prefix",
	Parse: func(value string) (string, error) {
		if !strings.HasSuffix(value, "/") {
			value += "/"
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

func mustProxyRegexpSet(value string) proxyRegexpSet {
	expressions, err := proxy.MakeRegexpArray(value)
	if err != nil {
		panic(err)
	}
	set := proxyRegexpSet{
		Source:      value,
		Expressions: expressions,
	}
	return set
}

type proxyConfig struct {
	Client           clientConfig
	Streams          genericiooptions.IOStreams
	Arguments        []string
	AcceptHosts      proxyRegexpSet
	AcceptPaths      proxyRegexpSet
	Address          string
	APIPrefix        string
	AppendServerPath bool
	DisableFilter    bool
	Keepalive        time.Duration
	Port             int
	RejectMethods    proxyRegexpSet
	RejectPaths      proxyRegexpSet
	UnixSocket       string
	Www              string
	WwwPrefix        string
}

func proxyCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[proxyConfig] {
	cmd := cli.Command[proxyConfig]{
		Name: "proxy",
		Description: cli.Description{
			Short: "Run a proxy to the Kubernetes API server",
			Long:  "Creates a proxy server or application-level gateway between localhost and the Kubernetes API server. It also allows serving static content over specified HTTP path. All incoming data enters through one port and gets forwarded to the remote Kubernetes API server port, except for the path matching the static content path.",
		},
		Examples: []cli.Example{
			{
				Description: "To proxy all of the Kubernetes API and nothing else",
				CommandLine: "kubectl proxy --api-prefix=/",
			},
			{
				Description: "To proxy only part of the Kubernetes API and also some static files\nYou can get pods info with 'curl localhost:8001/api/v1/pods'",
				CommandLine: "kubectl proxy --www=/my/files --www-prefix=/static/ --api-prefix=/api/",
			},
			{
				Description: "To proxy the entire Kubernetes API at a different root\nYou can get pods info with 'curl localhost:8001/custom/api/v1/pods'",
				CommandLine: "kubectl proxy --api-prefix=/custom/",
			},
			{
				Description: "Run a proxy to the Kubernetes API server on port 8011, serving static content from ./local/www/",
				CommandLine: "kubectl proxy --port=8011 --www=./local/www/",
			},
			{
				Description: "Run a proxy to the Kubernetes API server on an arbitrary local port\nThe chosen port for the server will be output to stdout",
				CommandLine: "kubectl proxy --port=0",
			},
			{
				Description: "Run a proxy to the Kubernetes API server, changing the API prefix to k8s-api\nThis makes e.g. the pods API available at localhost:8001/k8s-api/v1/pods/",
				CommandLine: "kubectl proxy --api-prefix=/k8s-api",
			},
		},
		Declare: func(cb *cli.CommandBuilder[proxyConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[proxyConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore())
			})
			acceptHostsFlag := cb.Flags.Value(&cb.Config.AcceptHosts, proxyRegexpSetLanguage)
			cb.Values.ResolveWith(acceptHostsFlag, cli.Value.KeepLast())
			acceptHostsFlag.Description("Regular expression for hosts that the proxy should accept.")
			cb.Values.Default(acceptHostsFlag, mustProxyRegexpSet("^localhost$,^127\\.0\\.0\\.1$,^\\[::1\\]$"))
			acceptHostsFlag.Long("accept-hosts")
			acceptPathsFlag := cb.Flags.Value(&cb.Config.AcceptPaths, proxyRegexpSetLanguage)
			cb.Values.ResolveWith(acceptPathsFlag, cli.Value.KeepLast())
			acceptPathsFlag.Description("Regular expression for paths that the proxy should accept.")
			cb.Values.Default(acceptPathsFlag, mustProxyRegexpSet("^.*"))
			acceptPathsFlag.Long("accept-paths")
			addressFlag := cb.Flags.String(&cb.Config.Address)
			cb.Values.ResolveWith(addressFlag, cli.Value.KeepLast())
			addressFlag.Description("The IP address on which to serve on.")
			cb.Values.Default(addressFlag, "127.0.0.1")
			addressFlag.Long("address")
			apiPrefixFlag := cb.Flags.Value(&cb.Config.APIPrefix, proxyPathPrefixLanguage)
			cb.Values.ResolveWith(apiPrefixFlag, cli.Value.KeepLast())
			apiPrefixFlag.Description("Prefix to serve the proxied API under.")
			cb.Values.Default(apiPrefixFlag, "/")
			apiPrefixFlag.Long("api-prefix")
			appendServerPathFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendServerPath))
			cb.Values.ResolveWith(appendServerPathFlag, cli.Value.KeepLast())
			appendServerPathFlag.Description("If true, enables automatic path appending of the kube context server path to each request.")
			appendServerPathFlag.Long("append-server-path").ImplicitValue(true)
			disableFilterFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.DisableFilter))
			cb.Values.ResolveWith(disableFilterFlag, cli.Value.KeepLast())
			disableFilterFlag.Description("If true, disable request filtering in the proxy. This is dangerous, and can leave you vulnerable to XSRF attacks, when used with an accessible port.")
			disableFilterFlag.Long("disable-filter").ImplicitValue(true)
			keepaliveFlag := cb.Flags.Value(&cb.Config.Keepalive, durationLanguage())
			cb.Values.ResolveWith(keepaliveFlag, cli.Value.KeepLast())
			keepaliveFlag.Description("keepalive specifies the keep-alive period for an active network connection. Set to 0 to disable keepalive.")
			keepaliveFlag.Long("keepalive")
			portFlag := cb.Flags.Int(&cb.Config.Port)
			cb.Values.ResolveWith(portFlag, cli.Value.KeepLast())
			portFlag.Description("The port on which to run the proxy. Set to 0 to pick a random port.")
			cb.Values.Default(portFlag, 8001)
			portFlag.Long("port")
			portFlag.Short('p')
			rejectMethodsFlag := cb.Flags.Value(&cb.Config.RejectMethods, proxyRegexpSetLanguage)
			cb.Values.ResolveWith(rejectMethodsFlag, cli.Value.KeepLast())
			rejectMethodsFlag.Description("Regular expression for HTTP methods that the proxy should reject (example --reject-methods='POST,PUT,PATCH'). ")
			cb.Values.Default(rejectMethodsFlag, mustProxyRegexpSet("^$"))
			rejectMethodsFlag.Long("reject-methods")
			rejectPathsFlag := cb.Flags.Value(&cb.Config.RejectPaths, proxyRegexpSetLanguage)
			cb.Values.ResolveWith(rejectPathsFlag, cli.Value.KeepLast())
			rejectPathsFlag.Description("Regular expression for paths that the proxy should reject. Paths specified here will be rejected even accepted by --accept-paths.")
			cb.Values.Default(rejectPathsFlag, mustProxyRegexpSet("^/api/.*/pods/.*/exec,^/api/.*/pods/.*/attach,^/api/.*/nodes/.*/proxy/run"))
			rejectPathsFlag.Long("reject-paths")
			unixSocketFlag := cb.Flags.String(&cb.Config.UnixSocket)
			cb.Values.ResolveWith(unixSocketFlag, cli.Value.KeepLast())
			unixSocketFlag.Description("Unix socket on which to run the proxy.")
			unixSocketFlag.Long("unix-socket")
			unixSocketFlag.Short('u')
			cb.Rule.When(cb.When.NotEqual(unixSocketFlag.Destination(), "")).Equal(portFlag.Destination(), 8001)
			wwwFlag := cb.Flags.String(&cb.Config.Www)
			cb.Values.ResolveWith(wwwFlag, cli.Value.KeepLast())
			wwwFlag.Description("Also serve static files from the given directory under the specified prefix.")
			wwwFlag.Long("www")
			wwwFlag.Short('w')
			wwwPrefixFlag := cb.Flags.Value(&cb.Config.WwwPrefix, proxyPathPrefixLanguage)
			cb.Values.ResolveWith(wwwPrefixFlag, cli.Value.KeepLast())
			wwwPrefixFlag.Description("Prefix to serve static files under, if static file directory is specified.")
			cb.Values.Default(wwwPrefixFlag, "/static/")
			wwwPrefixFlag.Long("www-prefix")
			wwwPrefixFlag.Short('P')
		},
		Run: runProxy,
	}
	return cmd
}

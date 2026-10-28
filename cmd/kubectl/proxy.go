package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type proxyConfig struct {
	Client           clientConfig
	Streams          genericiooptions.IOStreams
	Arguments        []string
	AcceptHosts      string
	AcceptPaths      string
	Address          string
	APIPrefix        string
	AppendServerPath bool
	DisableFilter    bool
	Keepalive        time.Duration
	Port             int
	RejectMethods    string
	RejectPaths      string
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

			acceptHostsFlag := cb.Flags.String(&cb.Config.AcceptHosts)
			acceptHostsFlag = acceptHostsFlag.Description("Regular expression for hosts that the proxy should accept.")
			cb.Values.Default(acceptHostsFlag, "^localhost$,^127\\.0\\.0\\.1$,^\\[::1\\]$")
			acceptHostsFlag.Long("accept-hosts")

			acceptPathsFlag := cb.Flags.String(&cb.Config.AcceptPaths)
			acceptPathsFlag = acceptPathsFlag.Description("Regular expression for paths that the proxy should accept.")
			cb.Values.Default(acceptPathsFlag, "^.*")
			acceptPathsFlag.Long("accept-paths")

			addressFlag := cb.Flags.String(&cb.Config.Address)
			addressFlag = addressFlag.Description("The IP address on which to serve on.")
			cb.Values.Default(addressFlag, "127.0.0.1")
			addressFlag.Long("address")

			apiPrefixFlag := cb.Flags.String(&cb.Config.APIPrefix)
			apiPrefixFlag = apiPrefixFlag.Description("Prefix to serve the proxied API under.")
			cb.Values.Default(apiPrefixFlag, "/")
			apiPrefixFlag.Long("api-prefix")

			appendServerPathFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AppendServerPath))
			appendServerPathFlag = appendServerPathFlag.Description("If true, enables automatic path appending of the kube context server path to each request.")
			appendServerPathFlag.Long("append-server-path").ImplicitValue(true)

			disableFilterFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.DisableFilter))
			disableFilterFlag = disableFilterFlag.Description("If true, disable request filtering in the proxy. This is dangerous, and can leave you vulnerable to XSRF attacks, when used with an accessible port.")
			disableFilterFlag.Long("disable-filter").ImplicitValue(true)

			keepaliveFlag := cb.Flags.Value(&cb.Config.Keepalive, durationLanguage())
			keepaliveFlag = keepaliveFlag.Description("keepalive specifies the keep-alive period for an active network connection. Set to 0 to disable keepalive.")
			keepaliveFlag.Long("keepalive")

			portFlag := cb.Flags.Int(&cb.Config.Port)
			portFlag = portFlag.Description("The port on which to run the proxy. Set to 0 to pick a random port.")
			cb.Values.Default(portFlag, 8001)
			portFlag.Long("port")
			portFlag.Short('p')

			rejectMethodsFlag := cb.Flags.String(&cb.Config.RejectMethods)
			rejectMethodsFlag = rejectMethodsFlag.Description("Regular expression for HTTP methods that the proxy should reject (example --reject-methods='POST,PUT,PATCH'). ")
			cb.Values.Default(rejectMethodsFlag, "^$")
			rejectMethodsFlag.Long("reject-methods")

			rejectPathsFlag := cb.Flags.String(&cb.Config.RejectPaths)
			rejectPathsFlag = rejectPathsFlag.Description("Regular expression for paths that the proxy should reject. Paths specified here will be rejected even accepted by --accept-paths.")
			cb.Values.Default(rejectPathsFlag, "^/api/.*/pods/.*/exec,^/api/.*/pods/.*/attach,^/api/.*/nodes/.*/proxy/run")
			rejectPathsFlag.Long("reject-paths")

			unixSocketFlag := cb.Flags.String(&cb.Config.UnixSocket)
			unixSocketFlag = unixSocketFlag.Description("Unix socket on which to run the proxy.")
			unixSocketFlag.Long("unix-socket")
			unixSocketFlag.Short('u')

			wwwFlag := cb.Flags.String(&cb.Config.Www)
			wwwFlag = wwwFlag.Description("Also serve static files from the given directory under the specified prefix.")
			wwwFlag.Long("www")
			wwwFlag.Short('w')

			wwwPrefixFlag := cb.Flags.String(&cb.Config.WwwPrefix)
			wwwPrefixFlag = wwwPrefixFlag.Description("Prefix to serve static files under, if static file directory is specified.")
			cb.Values.Default(wwwPrefixFlag, "/static/")
			wwwPrefixFlag.Long("www-prefix")
			wwwPrefixFlag.Short('P')
		},
		Run: runProxy,
	}
	return cmd
}

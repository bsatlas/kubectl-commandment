package kubectl

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/utils/net"
)

type portForwardPort struct {
	Local         string
	Remote        string
	ExplicitLocal bool
}

func (port portForwardPort) String() string {
	if port.ExplicitLocal {
		return port.Local + ":" + port.Remote
	}
	return port.Remote
}

type portForwardConfig struct {
	Client            clientConfig
	Streams           genericiooptions.IOStreams
	Resource          string
	Ports             []portForwardPort
	Address           []string
	PodRunningTimeout time.Duration
}

var portForwardPortLanguage = cli.ValueCodec[portForwardPort]{
	Name: "kubectl-port-forward-port",
	Parse: func(value string) (portForwardPort, error) {
		parts := strings.Split(value, ":")
		if len(parts) > 2 {
			return portForwardPort{}, cli.Rejection.Message(
				"port-forward-port",
				fmt.Sprintf("invalid port format %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		port := portForwardPort{
			Local:  parts[0],
			Remote: parts[0],
		}
		if len(parts) == 2 {
			port.Remote = parts[1]
			port.ExplicitLocal = true
		}
		if port.Remote == "" {
			return portForwardPort{}, cli.Rejection.Message(
				"port-forward-port",
				"remote port cannot be empty",
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		if port.ExplicitLocal && port.Local != "" && port.Local != port.Remote {
			if _, err := strconv.ParseUint(port.Local, 10, 16); err != nil {
				return portForwardPort{}, cli.Rejection.Message(
					"port-forward-port",
					fmt.Sprintf("error parsing local port %q: %s", port.Local, err),
					cli.Rejection.At(0, 0, len(value)),
				)
			}
		}
		if strings.Trim(port.Remote, "0123456789") == "" {
			number, err := strconv.ParseUint(port.Remote, 10, 16)
			if err != nil {
				return portForwardPort{}, cli.Rejection.Message(
					"port-forward-port",
					fmt.Sprintf("error parsing remote port %q: %s", port.Remote, err),
					cli.Rejection.At(0, 0, len(value)),
				)
			}
			if number == 0 {
				return portForwardPort{}, cli.Rejection.Message(
					"port-forward-port",
					"remote port must be > 0",
					cli.Rejection.At(0, 0, len(value)),
				)
			}
		}
		return port, nil
	},
	Format:   func(value portForwardPort) string { return value.String() },
	Snapshot: func(value portForwardPort) portForwardPort { return value },
}

var portForwardAddressLanguage = cli.ValueCodec[string]{
	Name: "kubectl-port-forward-address",
	Parse: func(value string) (string, error) {
		if value == "localhost" || net.ParseIPSloppy(value) != nil {
			return value, nil
		}
		return "", cli.Rejection.Message(
			"port-forward-address",
			fmt.Sprintf("%s is not a valid IP", value),
			cli.Rejection.At(0, 0, len(value)),
		)
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

func portForwardCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[portForwardConfig] {
	cmd := cli.Command[portForwardConfig]{
		Name: "port-forward",
		Description: cli.Description{
			Short: "Forward one or more local ports to a pod",
			Long:  "Forward one or more local ports to a pod.\n\n Use resource type/name such as deployment/mydeployment to select a pod. Resource type defaults to 'pod' if omitted.\n\n If there are multiple pods matching the criteria, a pod will be selected automatically. The forwarding session ends when the selected pod terminates, and a rerun of the command is needed to resume forwarding.",
		},
		Examples: []cli.Example{
			{
				Description: "Listen on ports 5000 and 6000 locally, forwarding data to/from ports 5000 and 6000 in the pod",
				CommandLine: "kubectl port-forward pod/mypod 5000 6000",
			},
			{
				Description: "Listen on ports 5000 and 6000 locally, forwarding data to/from ports 5000 and 6000 in a pod selected by the deployment",
				CommandLine: "kubectl port-forward deployment/mydeployment 5000 6000",
			},
			{
				Description: "Listen on port 8443 locally, forwarding to the targetPort of the service's port named \"https\" in a pod selected by the service",
				CommandLine: "kubectl port-forward service/myservice 8443:https",
			},
			{
				Description: "Listen on port 8888 locally, forwarding to 5000 in the pod",
				CommandLine: "kubectl port-forward pod/mypod 8888:5000",
			},
			{
				Description: "Listen on port 8888 on all addresses, forwarding to 5000 in the pod",
				CommandLine: "kubectl port-forward --address 0.0.0.0 pod/mypod 8888:5000",
			},
			{
				Description: "Listen on port 8888 on localhost and selected IP, forwarding to 5000 in the pod",
				CommandLine: "kubectl port-forward --address localhost,10.19.21.23 pod/mypod 8888:5000",
			},
			{
				Description: "Listen on a random port locally, forwarding to 5000 in the pod",
				CommandLine: "kubectl port-forward pod/mypod :5000",
			},
		},
		Declare: func(cb *cli.CommandBuilder[portForwardConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			ports := cb.Values.Sequence(&cb.Config.Ports, portForwardPortLanguage)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[portForwardConfig]) {
				completion := resourceAndPortArgumentCompleter(func(cfg *portForwardConfig) (clientConfig, []string) {
					arguments := make([]string, 0, 1+len(cfg.Ports))
					if cfg.Resource != "" {
						arguments = append(arguments, cfg.Resource)
					}
					for _, port := range cfg.Ports {
						arguments = append(arguments, port.String())
					}
					return cfg.Client, arguments
				})
				form.String(&form.Config.Resource, "TYPE/NAME").CompleteBy(completion).From(client)
				form.Sequence(ports, "PORT", cli.Count.OneOrMore()).CompleteBy(completion).From(client)
			})
			addressValue := cb.Values.Sequence(&cb.Config.Address, portForwardAddressLanguage)
			addressFlag := cb.Flags.Sequence(addressValue)
			addressFlag.DelimitedBy(",")
			addressFlag.Description("Addresses to listen on (comma separated). Only accepts IP addresses or localhost as a value. When localhost is supplied, kubectl will try to bind on both 127.0.0.1 and ::1 and will fail if neither of these addresses are available to bind.")
			cb.Values.Default(addressFlag, []string{"localhost"})
			addressFlag.Long("address")
			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			cb.Values.ResolveWith(podRunningTimeoutFlag, cli.Value.KeepLast())
			podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, time.Minute)
			podRunningTimeoutFlag.Long("pod-running-timeout")
			cb.Rule.GreaterThan(podRunningTimeoutFlag.Destination(), time.Duration(0))
		},
		Run: runPortForward,
	}
	return cmd
}

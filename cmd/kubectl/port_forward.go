package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type portForwardConfig struct {
	Client            clientConfig
	Streams           genericiooptions.IOStreams
	Arguments         []string
	Address           []string
	PodRunningTimeout time.Duration
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
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[portForwardConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceAndPortArgumentCompleter(
						func(cfg *portForwardConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
					)).
					From(client)
			})

			addressValue := cb.Values.Sequence(&cb.Config.Address, cli.Value.String())
			addressFlag := cb.Flags.Sequence(addressValue)
			addressFlag = addressFlag.DelimitedBy(",")
			addressFlag = addressFlag.Description("Addresses to listen on (comma separated). Only accepts IP addresses or localhost as a value. When localhost is supplied, kubectl will try to bind on both 127.0.0.1 and ::1 and will fail if neither of these addresses are available to bind.")
			cb.Values.Default(addressFlag, []string{"localhost"})
			addressFlag.Long("address")

			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			podRunningTimeoutFlag = podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, time.Minute)
			podRunningTimeoutFlag.Long("pod-running-timeout")
		},
		Run: runPortForward,
	}
	return cmd
}

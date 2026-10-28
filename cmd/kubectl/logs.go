package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type logsConfig struct {
	Client                       clientConfig
	Streams                      genericiooptions.IOStreams
	ContainerState               cli.OptionalValue[cli.ValueState]
	TailState                    cli.OptionalValue[cli.ValueState]
	Arguments                    []string
	AllContainers                bool
	AllPods                      bool
	Container                    string
	Follow                       bool
	IgnoreErrors                 bool
	InsecureSkipTLSVerifyBackend bool
	LimitBytes                   int64
	MaxLogRequests               int
	PodRunningTimeout            time.Duration
	Prefix                       bool
	Previous                     bool
	Selector                     string
	Since                        time.Duration
	SinceTime                    string
	Tail                         int64
	Timestamps                   bool
}

func logsCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[logsConfig] {
	cmd := cli.Command[logsConfig]{
		Name: "logs",
		Description: cli.Description{
			Short: "Print the logs for a container in a pod",
			Long:  "Print the logs for a container in a pod or specified resource. If the pod has only one container, the container name is optional.",
		},
		Examples: []cli.Example{
			{
				Description: "Return snapshot logs from pod nginx with only one container",
				CommandLine: "kubectl logs nginx",
			},
			{
				Description: "Return snapshot logs from pod nginx, prefixing each line with the source pod and container name",
				CommandLine: "kubectl logs nginx --prefix",
			},
			{
				Description: "Return snapshot logs from pod nginx, limiting output to 500 bytes",
				CommandLine: "kubectl logs nginx --limit-bytes=500",
			},
			{
				Description: "Return snapshot logs from pod nginx, waiting up to 20 seconds for it to start running.",
				CommandLine: "kubectl logs nginx --pod-running-timeout=20s",
			},
			{
				Description: "Return snapshot logs from pod nginx with multi containers",
				CommandLine: "kubectl logs nginx --all-containers=true",
			},
			{
				Description: "Return snapshot logs from all pods in the deployment nginx",
				CommandLine: "kubectl logs deployment/nginx --all-pods=true",
			},
			{
				Description: "Return snapshot logs from all containers in pods defined by label app=nginx",
				CommandLine: "kubectl logs -l app=nginx --all-containers=true",
			},
			{
				Description: "Return snapshot logs from all pods defined by label app=nginx, limiting concurrent log requests to 10 pods",
				CommandLine: "kubectl logs -l app=nginx --max-log-requests=10",
			},
			{
				Description: "Return snapshot of previous terminated ruby container logs from pod web-1",
				CommandLine: "kubectl logs -p -c ruby web-1",
			},
			{
				Description: "Begin streaming the logs from pod nginx, continuing even if errors occur",
				CommandLine: "kubectl logs nginx -f --ignore-errors=true",
			},
			{
				Description: "Begin streaming the logs of the ruby container in pod web-1",
				CommandLine: "kubectl logs -f -c ruby web-1",
			},
			{
				Description: "Begin streaming the logs from all containers in pods defined by label app=nginx",
				CommandLine: "kubectl logs -f -l app=nginx --all-containers=true",
			},
			{
				Description: "Display only the most recent 20 lines of output in pod nginx",
				CommandLine: "kubectl logs --tail=20 nginx",
			},
			{
				Description: "Show all logs from pod nginx written in the last hour",
				CommandLine: "kubectl logs --since=1h nginx",
			},
			{
				Description: "Show all logs with timestamps from pod nginx starting from August 30, 2024, at 06:00:00 UTC",
				CommandLine: "kubectl logs nginx --since-time=2024-08-30T06:00:00Z --timestamps=true",
			},
			{
				Description: "Show logs from a kubelet with an expired serving certificate",
				CommandLine: "kubectl logs --insecure-skip-tls-verify-backend nginx",
			},
			{
				Description: "Return snapshot logs from first container of a job named hello",
				CommandLine: "kubectl logs job/hello",
			},
			{
				Description: "Return snapshot logs from container nginx-1 of a deployment named nginx",
				CommandLine: "kubectl logs deployment/nginx -c nginx-1",
			},
		},
		Declare: func(cb *cli.CommandBuilder[logsConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[logsConfig]) {
				form.Sequence(arguments, "resource", cli.Count.Between(0, 2)).
					CompleteBy(podResourceAndContainerArgumentCompleter(
						func(cfg *logsConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
					)).
					From(client)
			})

			allContainersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllContainers))
			allContainersFlag = allContainersFlag.Description("Get all containers' logs in the pod(s).")
			allContainersFlag.Long("all-containers").ImplicitValue(true)

			allPodsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllPods))
			allPodsFlag = allPodsFlag.Description("Get logs from all pod(s). Sets prefix to true.")
			allPodsFlag.Long("all-pods").ImplicitValue(true)

			containerFlag := cb.Flags.String(&cb.Config.Container)
			containerFlag = containerFlag.Description("Print the logs of this container")
			containerFlag.Long("container")
			containerFlag.Short('c')
			cb.Access.ValueState(containerFlag, &cb.Config.ContainerState)

			followFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Follow))
			followFlag = followFlag.Description("Specify if the logs should be streamed.")
			followFlag.Long("follow").ImplicitValue(true)
			followFlag.Short('f').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			ignoreErrorsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.IgnoreErrors))
			ignoreErrorsFlag = ignoreErrorsFlag.Description("If watching / following pod logs, allow for any errors that occur to be non-fatal")
			ignoreErrorsFlag.Long("ignore-errors").ImplicitValue(true)

			insecureSkipTLSVerifyBackendFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.InsecureSkipTLSVerifyBackend))
			insecureSkipTLSVerifyBackendFlag = insecureSkipTLSVerifyBackendFlag.Description("Skip verifying the identity of the kubelet that logs are requested from.  In theory, an attacker could provide invalid log content back. You might want to use this if your kubelet serving certificates have expired.")
			insecureSkipTLSVerifyBackendFlag.Long("insecure-skip-tls-verify-backend").ImplicitValue(true)

			limitBytesFlag := cb.Flags.Value(&cb.Config.LimitBytes, int64Language())
			limitBytesFlag = limitBytesFlag.Description("Maximum bytes of logs to return. Defaults to no limit.")
			limitBytesFlag.Long("limit-bytes")

			maxLogRequestsFlag := cb.Flags.Int(&cb.Config.MaxLogRequests)
			maxLogRequestsFlag = maxLogRequestsFlag.Description("Specify maximum number of concurrent logs to follow when using by a selector. Defaults to 5.")
			cb.Values.Default(maxLogRequestsFlag, 5)
			maxLogRequestsFlag.Long("max-log-requests")

			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			podRunningTimeoutFlag = podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, 20*time.Second)
			podRunningTimeoutFlag.Long("pod-running-timeout")

			prefixFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Prefix))
			prefixFlag = prefixFlag.Description("Prefix each log line with the log source (pod name and container name)")
			prefixFlag.Long("prefix").ImplicitValue(true)

			previousFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Previous))
			previousFlag = previousFlag.Description("If true, print the logs for the previous instance of the container in a pod if it exists.")
			previousFlag.Long("previous").ImplicitValue(true)
			previousFlag.Short('p').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			sinceFlag := cb.Flags.Value(&cb.Config.Since, durationLanguage())
			sinceFlag = sinceFlag.Description("Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
			sinceFlag.Long("since")

			sinceTimeFlag := cb.Flags.String(&cb.Config.SinceTime)
			sinceTimeFlag = sinceTimeFlag.Description("Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
			sinceTimeFlag.Long("since-time")

			tailFlag := cb.Flags.Value(&cb.Config.Tail, int64Language())
			tailFlag = tailFlag.Description("Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines otherwise 10, if a selector is provided.")
			cb.Values.Default(tailFlag, int64(-1))
			tailFlag.Long("tail")
			cb.Access.ValueState(tailFlag, &cb.Config.TailState)

			timestampsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Timestamps))
			timestampsFlag = timestampsFlag.Description("Include timestamps on each line in the log output")
			timestampsFlag.Long("timestamps").ImplicitValue(true)
		},
		Run: runLogs,
	}
	return cmd
}

package kubectl

import (
	"context"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type logsConfig struct {
	Client                       clientConfig
	Streams                      genericiooptions.IOStreams
	Resource                     string
	InlineContainer              string
	AllContainers                bool
	AllPods                      bool
	Container                    string
	SelectedContainer            string
	Follow                       bool
	IgnoreErrors                 bool
	InsecureSkipTLSVerifyBackend bool
	LimitBytes                   int64
	MaxLogRequests               int
	PodRunningTimeout            time.Duration
	Prefix                       bool
	EffectivePrefix              bool
	Previous                     bool
	Selector                     string
	Since                        time.Duration
	SinceTime                    *time.Time
	Tail                         int64
	Timestamps                   bool
}

var logSinceTimeLanguage = cli.ValueCodec[*time.Time]{
	Name: "kubectl-rfc3339-time",
	Parse: func(value string) (*time.Time, error) {
		if value == "" {
			return nil, nil
		}
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, value)
		}
		if err != nil {
			return nil, cli.Rejection.Message(
				"rfc3339-time",
				err.Error(),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return &parsed, nil
	},
	Format: func(value *time.Time) string {
		if value == nil {
			return ""
		}
		return value.Format(time.RFC3339Nano)
	},
	Snapshot: func(value *time.Time) *time.Time {
		if value == nil {
			return nil
		}
		return new(*value)
	},
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
			resource := cb.Values.String(&cb.Config.Resource)
			inlineContainer := cb.Values.String(&cb.Config.InlineContainer)
			argumentCompletion := podResourceAndContainerArgumentCompleter(func(cfg *logsConfig) (clientConfig, []string) {
				arguments := make([]string, 0, 2)
				if cfg.Resource != "" {
					arguments = append(arguments, cfg.Resource)
				}
				if cfg.InlineContainer != "" {
					arguments = append(arguments, cfg.InlineContainer)
				}
				return cfg.Client, arguments
			})
			cb.Arguments.Form(func(*cli.ArgumentFormBuilder[logsConfig]) {})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[logsConfig]) {
				form.ValueFor(resource, "resource").CompleteBy(argumentCompletion).From(client)
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[logsConfig]) {
				form.ValueFor(resource, "resource").CompleteBy(argumentCompletion).From(client)
				form.ValueFor(inlineContainer, "container").CompleteBy(argumentCompletion).From(client)
			})
			allContainers := cb.Values.Bool(&cb.Config.AllContainers)
			allContainersFlag := cb.Flags.ValueFor(allContainers)
			cb.Values.ResolveWith(allContainersFlag, cli.Value.KeepLast())
			allContainersFlag.Description("Get all containers' logs in the pod(s).")
			allContainersFlag.Long("all-containers").ImplicitValue(true)
			allPods := cb.Values.Bool(&cb.Config.AllPods)
			allPodsFlag := cb.Flags.ValueFor(allPods)
			cb.Values.ResolveWith(allPodsFlag, cli.Value.KeepLast())
			allPodsFlag.Description("Get logs from all pod(s). Sets prefix to true.")
			allPodsFlag.Long("all-pods").ImplicitValue(true)
			containerFlag := cb.Flags.String(&cb.Config.Container)
			cb.Values.ResolveWith(containerFlag, cli.Value.KeepLast())
			containerFlag.Description("Print the logs of this container")
			containerFlag.Long("container")
			containerFlag.Short('c')
			followFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Follow))
			cb.Values.ResolveWith(followFlag, cli.Value.KeepLast())
			followFlag.Description("Specify if the logs should be streamed.")
			followFlag.Long("follow").ImplicitValue(true)
			followFlag.Short('f').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			ignoreErrorsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.IgnoreErrors))
			cb.Values.ResolveWith(ignoreErrorsFlag, cli.Value.KeepLast())
			ignoreErrorsFlag.Description("If watching / following pod logs, allow for any errors that occur to be non-fatal")
			ignoreErrorsFlag.Long("ignore-errors").ImplicitValue(true)
			insecureSkipTLSVerifyBackendFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.InsecureSkipTLSVerifyBackend))
			cb.Values.ResolveWith(insecureSkipTLSVerifyBackendFlag, cli.Value.KeepLast())
			insecureSkipTLSVerifyBackendFlag.Description("Skip verifying the identity of the kubelet that logs are requested from.  In theory, an attacker could provide invalid log content back. You might want to use this if your kubelet serving certificates have expired.")
			insecureSkipTLSVerifyBackendFlag.Long("insecure-skip-tls-verify-backend").ImplicitValue(true)
			limitBytesFlag := cb.Flags.Value(&cb.Config.LimitBytes, int64Language())
			cb.Values.ResolveWith(limitBytesFlag, cli.Value.KeepLast())
			limitBytesFlag.Description("Maximum bytes of logs to return. Defaults to no limit.")
			limitBytesFlag.Long("limit-bytes")
			maxLogRequestsFlag := cb.Flags.Int(&cb.Config.MaxLogRequests)
			cb.Values.ResolveWith(maxLogRequestsFlag, cli.Value.KeepLast())
			maxLogRequestsFlag.Description("Specify maximum number of concurrent logs to follow when using by a selector. Defaults to 5.")
			cb.Values.Default(maxLogRequestsFlag, 5)
			maxLogRequestsFlag.Long("max-log-requests")
			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			cb.Values.ResolveWith(podRunningTimeoutFlag, cli.Value.KeepLast())
			podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, 20*time.Second)
			podRunningTimeoutFlag.Long("pod-running-timeout")
			prefix := cb.Values.Bool(&cb.Config.Prefix)
			prefixFlag := cb.Flags.ValueFor(prefix)
			cb.Values.ResolveWith(prefixFlag, cli.Value.KeepLast())
			prefixFlag.Description("Prefix each log line with the log source (pod name and container name)")
			prefixFlag.Long("prefix").ImplicitValue(true)
			previousFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Previous))
			cb.Values.ResolveWith(previousFlag, cli.Value.KeepLast())
			previousFlag.Description("If true, print the logs for the previous instance of the container in a pod if it exists.")
			previousFlag.Long("previous").ImplicitValue(true)
			previousFlag.Short('p').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')
			sinceFlag := cb.Flags.Value(&cb.Config.Since, durationLanguage())
			cb.Values.ResolveWith(sinceFlag, cli.Value.KeepLast())
			sinceFlag.Description("Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
			sinceFlag.Long("since")
			sinceTimeFlag := cb.Flags.Value(&cb.Config.SinceTime, logSinceTimeLanguage)
			cb.Values.ResolveWith(sinceTimeFlag, cli.Value.KeepLast())
			sinceTimeFlag.Description("Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
			sinceTimeFlag.Long("since-time")
			tailFlag := cb.Flags.Value(&cb.Config.Tail, int64Language())
			cb.Values.ResolveWith(tailFlag, cli.Value.KeepLast())
			tailFlag.Description("Lines of recent log file to display. Defaults to -1 with no selector, showing all log lines otherwise 10, if a selector is provided.")
			cb.Values.ComputedDefault(
				tailFlag.Destination(),
				cli.Value.ComputedBy(func(_ context.Context, cfg *logsConfig, _ cli.Dependencies) (int64, error) {
					if cfg.Selector != "" {
						return 10, nil
					}
					return -1, nil
				}).From(selectorFlag.OptionalDependency()),
			)
			tailFlag.Long("tail")
			cb.Values.Derive(
				&cb.Config.SelectedContainer,
				cli.Value.ComputedBy(func(_ context.Context, cfg *logsConfig, _ cli.Dependencies) (string, error) {
					if cfg.InlineContainer != "" {
						return cfg.InlineContainer, nil
					}
					return cfg.Container, nil
				}).From(inlineContainer.OptionalDependency(), containerFlag.OptionalDependency()),
			)
			cb.Values.Derive(
				&cb.Config.EffectivePrefix,
				cli.Value.ComputedBy(func(_ context.Context, cfg *logsConfig, _ cli.Dependencies) (bool, error) {
					return cfg.AllPods || cfg.Prefix, nil
				}).From(allPods.OptionalDependency(), prefix.OptionalDependency()),
			)
			resourceProvided := cb.Rule.ContributedBy(resource, cli.ValueProducerKindArgument)
			inlineContainerProvided := cb.Rule.ContributedBy(inlineContainer, cli.ValueProducerKindArgument)
			cb.Rule.Unless(resourceProvided).Require(selectorFlag.Destination())
			cb.Rule.Unless(resourceProvided).NotEmpty(selectorFlag.Destination())
			cb.Rule.When(cb.Rule.All(resourceProvided, cb.Rule.Not(inlineContainerProvided))).Equal(selectorFlag.Destination(), "")
			cb.Rule.When(inlineContainerProvided).Forbid(containerFlag.Destination())
			cb.Rule.When(cb.When.Equal(allContainers, true)).Equal(containerFlag.Destination(), "")
			cb.Rule.When(cb.When.Equal(allContainers, true)).Equal(inlineContainer, "")
			cb.Rule.Check("kubectl-logs-since-exclusive", func(_ context.Context, cfg *logsConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Since == 0 || cfg.SinceTime == nil, nil
			}).
				DependsOn(sinceFlag.OptionalDependency()).
				DependsOn(sinceTimeFlag.OptionalDependency())
			cb.Rule.AtLeast(limitBytesFlag.Destination(), int64(0))
			cb.Rule.Check("kubectl-logs-since-nonnegative-seconds", func(_ context.Context, cfg *logsConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Since == 0 {
					return true, nil
				}
				return int64(cfg.Since.Round(time.Second).Seconds()) >= 0, nil
			}).DependsOn(sinceFlag.OptionalDependency())
			cb.Rule.AtLeast(tailFlag.Destination(), int64(-1))
			cb.Rule.GreaterThan(podRunningTimeoutFlag.Destination(), time.Duration(0))
			timestampsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Timestamps))
			cb.Values.ResolveWith(timestampsFlag, cli.Value.KeepLast())
			timestampsFlag.Description("Include timestamps on each line in the log output")
			timestampsFlag.Long("timestamps").ImplicitValue(true)
		},
		Run: runLogs,
	}
	return cmd
}

package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

var drainPodSelectorLanguage = cli.ValueCodec[string]{
	Name: "pod-selector",
	Parse: func(value string) (string, error) {
		if _, err := labels.Parse(value); err != nil {
			return "", cli.Rejection.Message(
				"pod-selector",
				"--pod-selector=<pod_selector> must be a valid label selector",
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

type drainConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Arguments                []string
	ChunkSize                int64
	DeleteEmptydirData       bool
	DisableEviction          bool
	DryRun                   string
	Force                    bool
	GracePeriod              int
	IgnoreDaemonsets         bool
	PodSelector              string
	Selector                 string
	SkipWaitForDeleteTimeout int
	Timeout                  time.Duration
}

func drainCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[drainConfig] {
	cmd := cli.Command[drainConfig]{
		Name: "drain",
		Description: cli.Description{
			Short: "Drain node in preparation for maintenance",
			Long:  "Drain node in preparation for maintenance.\n\n The given node will be marked unschedulable to prevent new pods from arriving. 'drain' evicts the pods if the API server supports https://kubernetes.io/docs/concepts/workloads/pods/disruptions/ eviction https://kubernetes.io/docs/concepts/workloads/pods/disruptions/ . Otherwise, it will use normal DELETE to delete the pods. The 'drain' evicts or deletes all pods except mirror pods (which cannot be deleted through the API server).  If there are daemon set-managed pods, drain will not proceed without --ignore-daemonsets, and regardless it will not delete any daemon set-managed pods, because those pods would be immediately replaced by the daemon set controller, which ignores unschedulable markings.  If there are any pods that are neither mirror pods nor managed by a replication controller, replica set, daemon set, stateful set, or job, then drain will not delete any pods unless you use --force.  --force will also allow deletion to proceed if the managing resource of one or more pods is missing.\n\n 'drain' waits for graceful termination. You should not operate on the machine until the command completes.\n\n When you are ready to put the node back into service, use kubectl uncordon, which will make the node schedulable again.\n\nhttps://kubernetes.io/images/docs/kubectl_drain.svg Workflowhttps://kubernetes.io/images/docs/kubectl_drain.svg",
		},
		Examples: []cli.Example{
			{
				Description: "Drain node \"foo\", even if there are pods not managed by a replication controller, replica set, job, daemon set, or stateful set on it",
				CommandLine: "kubectl drain foo --force",
			},
			{
				Description: "As above, but abort if there are pods not managed by a replication controller, replica set, job, daemon set, or stateful set, and use a grace period of 15 minutes",
				CommandLine: "kubectl drain foo --grace-period=900",
			},
		},
		Declare: func(cb *cli.CommandBuilder[drainConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[drainConfig]) {
				form.Sequence(arguments, "node", cli.Count.ZeroOrMore()).
					CompleteBy(fixedResourceArgumentCompleter(
						func(cfg *drainConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						"node",
					)).
					From(client)
			})

			chunkSizeFlag := cb.Flags.Value(&cb.Config.ChunkSize, int64Language())
			chunkSizeFlag.Description("Return large lists in chunks rather than all at once. Pass 0 to disable.")
			cb.Values.Default(chunkSizeFlag, int64(500))
			chunkSizeFlag.Long("chunk-size")

			deleteEmptydirDataFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.DeleteEmptydirData))
			deleteEmptydirDataFlag.Description("Continue even if there are pods using emptyDir (local data that will be deleted when the node is drained).")
			deleteEmptydirDataFlag.Long("delete-emptydir-data").ImplicitValue(true)

			disableEvictionFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.DisableEviction))
			disableEvictionFlag.Description("Force drain to use delete, even if eviction is supported. This will bypass checking PodDisruptionBudgets, use with caution.")
			disableEvictionFlag.Long("disable-eviction").ImplicitValue(true)

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlag.Long("dry-run").ImplicitValue("unchanged")

			forceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force))
			forceFlag.Description("Continue even if there are pods that do not declare a controller.")
			forceFlag.Long("force").ImplicitValue(true)

			gracePeriodFlag := cb.Flags.Int(&cb.Config.GracePeriod)
			gracePeriodFlag.Description("Period of time in seconds given to each pod to terminate gracefully. If negative, the default value specified in the pod will be used.")
			cb.Values.Default(gracePeriodFlag, -1)
			gracePeriodFlag.Long("grace-period")

			ignoreDaemonsetsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.IgnoreDaemonsets))
			ignoreDaemonsetsFlag.Description("Ignore DaemonSet-managed pods.")
			ignoreDaemonsetsFlag.Long("ignore-daemonsets").ImplicitValue(true)

			podSelectorFlag := cb.Flags.Value(&cb.Config.PodSelector, drainPodSelectorLanguage)
			podSelectorFlag.Description("Label selector to filter pods on the node")
			podSelectorFlag.Long("pod-selector")

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag.Description("Selector (label query) to filter nodes on.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			cb.Rule.AtLeastOne(arguments, selectorFlag)
			cb.Rule.When(cb.When.NotEqual(selectorFlag.Destination(), "")).Forbid(arguments)

			skipWaitForDeleteTimeoutFlag := cb.Flags.Int(&cb.Config.SkipWaitForDeleteTimeout)
			skipWaitForDeleteTimeoutFlag.Description("If pod DeletionTimestamp older than N seconds, skip waiting for the pod. Seconds must be greater than 0 to skip.")
			skipWaitForDeleteTimeoutFlag.Long("skip-wait-for-delete-timeout")

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag.Description("The length of time to wait before giving up, zero means infinite")
			timeoutFlag.Long("timeout")
		},
		Run: runDrain,
	}
	return cmd
}

package kubectl

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutStatusConfig struct {
	Client    clientConfig
	Streams   genericiooptions.IOStreams
	Resources resourceArguments
	Filename  filenameOptions
	Revision  int64
	Selector  string
	Timeout   time.Duration
	Watch     bool
}

func rolloutStatusCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[rolloutStatusConfig] {
	cmd := cli.Command[rolloutStatusConfig]{
		Name: "status",
		Description: cli.Description{
			Short: "Show the status of the rollout",
			Long:  "Show the status of the rollout.\n\nBy default 'rollout status' will watch the status of the latest rollout until it's done. If you don't want to wait for the rollout to finish then you can use --watch=false. Note that if a new rollout starts in-between, then 'rollout status' will continue watching the latest revision. If you want to pin to a specific revision and abort if it is rolled over by another revision, use --revision=N where N is the revision you need to watch for.",
		},
		Examples: []cli.Example{
			{
				Description: "Watch the rollout status of a deployment",
				CommandLine: "kubectl rollout status deployment/nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[rolloutStatusConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			resources := declareResourceArguments(
				cb,
				&cb.Config.Resources,
				client,
				func(cfg *rolloutStatusConfig) (clientConfig, resourceArguments) { return cfg.Client, cfg.Resources },
				[]string{"deployment", "daemonset", "statefulset"},
			)
			files := declareFilenameOptions(cb, &cb.Config.Filename)

			revisionFlag := cb.Flags.Value(&cb.Config.Revision, int64Language()).Description("Pin to a specific revision for showing its status. Defaults to 0 (last revision).")
			revisionFlag.Long("revision")

			selectorFlag := cb.Flags.String(&cb.Config.Selector).Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			cb.Flags.Value(&cb.Config.Timeout, durationLanguage()).Description("The length of time to wait before ending watch, zero means never. Any other values should contain a corresponding time unit (e.g. 1s, 2m, 3h).").Long("timeout")

			watch := cb.Values.Bool(&cb.Config.Watch)
			cb.Values.Default(watch, true)
			watchFlag := cb.Flags.ValueFor(watch).Description("Watch the status of the rollout until it's done.")
			watchFlag.Long("watch").ImplicitValue(true)
			watchFlag.Short('w').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			requireResourceOrFilenameSource(cb, resources, files)
			cb.Rule.AtLeast(revisionFlag.Destination(), int64(0))
		},
		Run: runRolloutStatus,
	}
	return cmd
}

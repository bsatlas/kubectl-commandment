package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type rolloutStatusConfig struct {
	Client    clientConfig
	Streams   genericiooptions.IOStreams
	Arguments []string
	Filename  []string
	Kustomize string
	Recursive bool
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
			Long:  "Show the status of the rollout.\n\n By default 'rollout status' will watch the status of the latest rollout until it's done. If you don't want to wait for the rollout to finish then you can use --watch=false. Note that if a new rollout starts in-between, then 'rollout status' will continue watching the latest revision. If you want to pin to a specific revision and abort if it is rolled over by another revision, use --revision=N where N is the revision you need to watch for.",
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

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[rolloutStatusConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *rolloutStatusConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						[]string{"deployment", "daemonset", "statefulset"},
						resourceArgumentSingle,
					)).
					From(client)
			})

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("Filename, directory, or URL to files identifying the resource to get from a server.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process the kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			revisionFlag := cb.Flags.Value(&cb.Config.Revision, int64Language())
			revisionFlag = revisionFlag.Description("Pin to a specific revision for showing its status. Defaults to 0 (last revision).")
			revisionFlag.Long("revision")

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag = timeoutFlag.Description("The length of time to wait before ending watch, zero means never. Any other values should contain a corresponding time unit (e.g. 1s, 2m, 3h).")
			timeoutFlag.Long("timeout")

			watchFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Watch))
			watchFlag = watchFlag.Description("Watch the status of the rollout until it's done.")
			cb.Values.Default(watchFlag, true)
			watchFlag.Long("watch").ImplicitValue(true)
			watchFlag.Short('w').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
		},
		Run: runRolloutStatus,
	}
	return cmd
}

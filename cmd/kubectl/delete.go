package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type deleteConfig struct {
	Client              clientConfig
	Streams             genericiooptions.IOStreams
	IgnoreNotFoundState cli.OptionalValue[cli.ValueState]
	Arguments           []string
	All                 bool
	AllNamespaces       bool
	Cascade             string
	DryRun              string
	FieldSelector       string
	Filename            []string
	Force               bool
	GracePeriod         int
	IgnoreNotFound      bool
	Interactive         bool
	Kustomize           string
	Now                 bool
	Output              string
	Raw                 string
	Recursive           bool
	Selector            string
	Timeout             time.Duration
	Wait                bool
}

func deleteCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[deleteConfig] {
	cmd := cli.Command[deleteConfig]{
		Name: "delete",
		Description: cli.Description{
			Short: "Delete resources by file names, stdin, resources and names, or by resources and label selector",
			Long:  "Delete resources by file names, stdin, resources and names, or by resources and label selector.\n\n JSON and YAML formats are accepted. Only one type of argument may be specified: file names, resources and names, or resources and label selector.\n\n Some resources, such as pods, support graceful deletion. These resources define a default period before they are forcibly terminated (the grace period) but you may override that value with the --grace-period flag, or pass --now to set a grace-period of 1. Because these resources often represent entities in the cluster, deletion may not be acknowledged immediately. If the node hosting a pod is down or cannot reach the API server, termination may take significantly longer than the grace period. To force delete a resource, you must specify the --force flag. Note: only a subset of resources support graceful deletion. In absence of the support, the --grace-period flag is ignored.\n\n IMPORTANT: Force deleting pods does not wait for confirmation that the pod's processes have been terminated, which can leave those processes running until the node detects the deletion and completes graceful deletion. If your processes use shared storage or talk to a remote API and depend on the name of the pod to identify themselves, force deleting those pods may result in multiple processes running on different machines using the same identification which may lead to data corruption or inconsistency. Only force delete pods when you are sure the pod is terminated, or if your application can tolerate multiple copies of the same pod running at once. Also, if you force delete pods, the scheduler may place new pods on those nodes before the node has released those resources and causing those pods to be evicted immediately.\n\n Note that the delete command does NOT do resource version checks, so if someone submits an update to a resource right when you submit a delete, their update will be lost along with the rest of the resource.\n\n After a CustomResourceDefinition is deleted, invalidation of discovery cache may take up to 6 hours. If you don't want to wait, you might want to run \"kubectl api-resources\" to refresh the discovery cache.",
		},
		Examples: []cli.Example{
			{
				Description: "Delete a pod using the type and name specified in pod.json",
				CommandLine: "kubectl delete -f ./pod.json",
			},
			{
				Description: "Delete resources from a directory containing kustomization.yaml - e.g. dir/kustomization.yaml",
				CommandLine: "kubectl delete -k dir",
			},
			{
				Description: "Delete resources from all files that end with '.json'",
				CommandLine: "kubectl delete -f '*.json'",
			},
			{
				Description: "Delete a pod based on the type and name in the JSON passed into stdin",
				CommandLine: "cat pod.json | kubectl delete -f -",
			},
			{
				Description: "Delete pods and services with same names \"baz\" and \"foo\"",
				CommandLine: "kubectl delete pod,service baz foo",
			},
			{
				Description: "Delete pods and services with label name=myLabel",
				CommandLine: "kubectl delete pods,services -l name=myLabel",
			},
			{
				Description: "Delete a pod with minimal delay",
				CommandLine: "kubectl delete pod foo --now",
			},
			{
				Description: "Force delete a pod on a dead node",
				CommandLine: "kubectl delete pod foo --force",
			},
			{
				Description: "Delete all pods",
				CommandLine: "kubectl delete pods --all",
			},
			{
				Description: "Delete all pods only if the user confirms the deletion",
				CommandLine: "kubectl delete pods --all --interactive",
			},
		},
		Declare: func(cb *cli.CommandBuilder[deleteConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[deleteConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore()).
					CompleteBy(resourceArgumentCompleter(
						func(cfg *deleteConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
			})

			allFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.All))
			allFlag = allFlag.Description("Delete all resources, in the namespace of the specified resource types.")
			allFlag.Long("all").ImplicitValue(true)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag = allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			cascadeFlag := cb.Flags.String(&cb.Config.Cascade)
			cascadeFlag = cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cb.Values.Default(cascadeFlag, "background")
			cascadeFlagLong := cascadeFlag.Long("cascade")
			cascadeFlagLong.ImplicitValue("background")

			dryRunFlag := cb.Flags.String(&cb.Config.DryRun)
			dryRunFlag = dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, "none")
			dryRunFlagLong := dryRunFlag.Long("dry-run")
			dryRunFlagLong.ImplicitValue("unchanged")

			fieldSelectorFlag := cb.Flags.String(&cb.Config.FieldSelector)
			fieldSelectorFlag = fieldSelectorFlag.Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("containing the resource to delete.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			forceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force))
			forceFlag = forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			gracePeriodFlag := cb.Flags.Int(&cb.Config.GracePeriod)
			gracePeriodFlag = gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			cb.Values.Default(gracePeriodFlag, -1)
			gracePeriodFlag.Long("grace-period")

			ignoreNotFoundFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.IgnoreNotFound))
			ignoreNotFoundFlag = ignoreNotFoundFlag.Description("Treat \"resource not found\" as a successful delete. Defaults to \"true\" when --all is specified.")
			ignoreNotFoundFlag.Long("ignore-not-found").ImplicitValue(true)
			cb.Access.ValueState(ignoreNotFoundFlag, &cb.Config.IgnoreNotFoundState)

			interactiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Interactive))
			interactiveFlag = interactiveFlag.Description("If true, delete resource only when user confirms.")
			interactiveFlag.Long("interactive").ImplicitValue(true)
			interactiveFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			kustomizeFlag = kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k')

			nowFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Now))
			nowFlag = nowFlag.Description("If true, resources are signaled for immediate shutdown (same as --grace-period=1).")
			nowFlag.Long("now").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("Output mode. Use \"-o name\" for shorter output (resource/name).")
			outputFlag.Long("output")
			outputFlag.Short('o')

			rawFlag := cb.Flags.String(&cb.Config.Raw)
			rawFlag = rawFlag.Description("Raw URI to DELETE to the server.  Uses the transport specified by the kubeconfig file.")
			rawFlag.Long("raw")

			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			recursiveFlag = recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			selectorFlag := cb.Flags.String(&cb.Config.Selector)
			selectorFlag = selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l')

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			timeoutFlag = timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")

			waitFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Wait))
			waitFlag = waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			cb.Values.Default(waitFlag, true)
			waitFlag.Long("wait").ImplicitValue(true)
		},
		Run: runDelete,
	}
	return cmd
}

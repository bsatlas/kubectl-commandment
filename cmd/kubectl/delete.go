package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type deletionPropagation struct {
	Propagation       v1.DeletionPropagation
	DeprecatedBoolean string
}

func (cascade deletionPropagation) flagValue() string {
	return strings.ToLower(string(cascade.Propagation))
}

func (cascade deletionPropagation) deprecationWarning() string {
	if cascade.DeprecatedBoolean == "" {
		return ""
	}
	replacement := "orphan"
	if cascade.Propagation == v1.DeletePropagationBackground {
		replacement = "background"
	}
	return fmt.Sprintf("warning: --cascade=%v is deprecated (boolean value) and can be replaced with --cascade=%s.", cascade.DeprecatedBoolean, replacement)
}

var deletionPropagationLanguage = cli.ValueCodec[deletionPropagation]{
	Name: "kubectl-delete-cascade",
	Parse: func(value string) (deletionPropagation, error) {
		if boolean, err := strconv.ParseBool(value); err == nil {
			propagation := v1.DeletePropagationOrphan
			if boolean {
				propagation = v1.DeletePropagationBackground
			}
			cascade := deletionPropagation{
				Propagation:       propagation,
				DeprecatedBoolean: value,
			}
			return cascade, nil
		}

		var propagation v1.DeletionPropagation
		switch value {
		case "background":
			propagation = v1.DeletePropagationBackground
		case "foreground":
			propagation = v1.DeletePropagationForeground
		case "orphan":
			propagation = v1.DeletePropagationOrphan
		default:
			message := fmt.Sprintf(`invalid cascade value (%v). Must be "background", "foreground", or "orphan"`, value)
			return deletionPropagation{}, cli.Rejection.Message(
				"kubectl-delete-cascade",
				message,
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		cascade := deletionPropagation{
			Propagation: propagation,
		}
		return cascade, nil
	},
	Format: func(value deletionPropagation) string { return value.flagValue() },
	Snapshot: func(value deletionPropagation) deletionPropagation {
		return value
	},
}

type deleteConfig struct {
	Client         clientConfig
	Streams        genericiooptions.IOStreams
	Arguments      []string
	All            bool
	AllNamespaces  bool
	Cascade        deletionPropagation
	DryRun         util.DryRunStrategy
	FieldSelector  string
	Filename       filenameOptions
	Force          bool
	GracePeriod    int
	RequestedGrace int
	IgnoreNotFound bool
	Interactive    bool
	Now            bool
	Output         string
	Raw            string
	Selector       string
	Timeout        time.Duration
	Wait           bool
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

			all := cb.Values.Bool(&cb.Config.All)
			allFlag := cb.Flags.ValueFor(all)
			cb.Values.ResolveWith(allFlag, cli.Value.KeepLast())
			allFlag.Description("Delete all resources, in the namespace of the specified resource types.")
			allFlag.Long("all").ImplicitValue(true)

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			cb.Values.ResolveWith(allNamespacesFlag, cli.Value.KeepLast())
			allNamespacesFlag.Description("If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			cascade := cb.Values.Value(&cb.Config.Cascade, deletionPropagationLanguage)
			backgroundCascade := deletionPropagation{
				Propagation: v1.DeletePropagationBackground,
			}
			cb.Values.Default(cascade, backgroundCascade)
			cascadeFlag := cb.Flags.ValueFor(cascade)
			cb.Values.ResolveWith(cascadeFlag, cli.Value.KeepLast())
			cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cascadeFlag.Long("cascade").ImplicitValue(backgroundCascade)

			dryRun := cb.Values.Value(&cb.Config.DryRun, dryRunValueLanguage)
			dryRunFlag := cb.Flags.ValueFor(dryRun)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRun, util.DryRunNone)
			dryRunFlag.Long("dry-run").ImplicitValue(util.DryRunClient)

			selector := cb.Values.String(&cb.Config.Selector)
			selectorFlag := cb.Flags.ValueFor(selector)
			cb.Values.ResolveWith(selectorFlag, cli.Value.KeepLast())
			selectorFlag.Description("Selector (label query) to filter on, supports '=', '==', '!=', 'in', 'notin'.(e.g. -l key1=value1,key2=value2,key3 in (value3)). Matching objects must satisfy all of the specified label constraints.")
			selectorFlag.Long("selector")
			selectorFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			fieldSelector := cb.Values.String(&cb.Config.FieldSelector)
			fieldSelectorFlag := cb.Flags.ValueFor(fieldSelector)
			cb.Values.ResolveWith(fieldSelectorFlag, cli.Value.KeepLast())
			fieldSelectorFlag.Description("Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
			fieldSelectorFlag.Long("field-selector")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename.Groups, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("containing the resource to delete.")
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			force := cb.Values.Bool(&cb.Config.Force)
			forceFlag := cb.Flags.ValueFor(force)
			cb.Values.ResolveWith(forceFlag, cli.Value.KeepLast())
			forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)

			requestedGrace := cb.Values.Int(&cb.Config.RequestedGrace)
			cb.Values.Default(requestedGrace, -1)
			gracePeriodFlag := cb.Flags.ValueFor(requestedGrace)
			cb.Values.ResolveWith(gracePeriodFlag, cli.Value.KeepLast())
			gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			gracePeriodFlag.Long("grace-period")

			ignoreNotFound := cb.Values.Bool(&cb.Config.IgnoreNotFound)
			cb.Values.ComputedDefault(ignoreNotFound, cli.Value.ComputedBy(func(_ context.Context, _ *deleteConfig, dependencies cli.Dependencies) (bool, error) {
				deleteAll, _ := dependencies.Get(all)
				selector, _ := dependencies.Get(selector)
				fieldSelector, _ := dependencies.Get(fieldSelector)
				return deleteAll || selector != "" || fieldSelector != "", nil
			}).From(all.OptionalDependency(), selector.OptionalDependency(), fieldSelector.OptionalDependency()))
			ignoreNotFoundFlag := cb.Flags.ValueFor(ignoreNotFound)
			cb.Values.ResolveWith(ignoreNotFoundFlag, cli.Value.KeepLast())
			ignoreNotFoundFlag.Description("Treat \"resource not found\" as a successful delete. Defaults to \"true\" when --all is specified.")
			ignoreNotFoundFlag.Long("ignore-not-found").ImplicitValue(true)

			interactive := cb.Values.Bool(&cb.Config.Interactive)
			interactiveFlag := cb.Flags.ValueFor(interactive)
			cb.Values.ResolveWith(interactiveFlag, cli.Value.KeepLast())
			interactiveFlag.Description("If true, delete resource only when user confirms.")
			interactiveFlag.Long("interactive").ImplicitValue(true)
			interactiveFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			kustomize := cb.Values.String(&cb.Config.Filename.Kustomize)
			kustomizeFlag := cb.Flags.ValueFor(kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			now := cb.Values.Bool(&cb.Config.Now)
			nowFlag := cb.Flags.ValueFor(now)
			cb.Values.ResolveWith(nowFlag, cli.Value.KeepLast())
			nowFlag.Description("If true, resources are signaled for immediate shutdown (same as --grace-period=1).")
			nowFlag.Long("now").ImplicitValue(true)

			output := cb.Values.String(&cb.Config.Output)
			outputFlag := cb.Flags.ValueFor(output)
			cb.Values.ResolveWith(outputFlag, cli.Value.KeepLast())
			outputFlag.Description("Output mode. Use \"-o name\" for shorter output (resource/name).")
			outputFlag.Long("output")
			outputFlag.Short('o').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)

			raw := cb.Values.String(&cb.Config.Raw)
			rawFlag := cb.Flags.ValueFor(raw)
			cb.Values.ResolveWith(rawFlag, cli.Value.KeepLast())
			rawFlag.Description("Raw URI to DELETE to the server.  Uses the transport specified by the kubeconfig file.")
			rawFlag.Long("raw")

			recursive := cb.Values.Bool(&cb.Config.Filename.Recursive)
			recursiveFlag := cb.Flags.ValueFor(recursive)
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			deleteAll := cb.When.Equal(all, true)
			cb.Rule.When(deleteAll).Equal(selector, "")
			cb.Rule.When(deleteAll).Equal(fieldSelector, "")
			deleteNow := cb.When.Equal(now, true)
			cb.Rule.When(deleteNow).Equal(requestedGrace, -1)
			cb.Rule.When(deleteNow).Equal(force, false)
			cb.Rule.When(cb.When.Equal(force, true)).AtMost(requestedGrace, 0)
			rawDelete := cb.When.NotEqual(raw, "")
			cb.Rule.Check("kubectl-delete-source", func(_ context.Context, cfg *deleteConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw != "" {
					return true, nil
				}
				hasFilename := len(cfg.Filename.Filenames()) > 0
				hasKustomize := cfg.Filename.Kustomize != ""
				if hasFilename && hasKustomize {
					return false, nil
				}
				return len(cfg.Arguments) > 0 || hasFilename || hasKustomize, nil
			}).
				DependsOn(raw.OptionalDependency()).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency()).
				DependsOn(kustomize.OptionalDependency())
			cb.Rule.When(cb.When.Equal(recursive, true)).Equal(kustomize, "")
			cb.Rule.When(rawDelete).Equal(recursive, false)
			cb.Rule.When(rawDelete).Equal(output, "")
			cb.Rule.When(rawDelete).Equal(interactive, false)
			cb.Rule.Check("kubectl-delete-raw-filename-count", func(_ context.Context, cfg *deleteConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Raw == "" || len(cfg.Filename.Filenames()) <= 1, nil
			}).
				DependsOn(raw.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			cb.Rule.In(output, "", "name")
			cb.Rule.Check("kubectl-delete-raw-uri", func(_ context.Context, cfg *deleteConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" {
					return true, nil
				}
				_, err := url.ParseRequestURI(cfg.Raw)
				return err == nil, nil
			}).DependsOn(raw.OptionalDependency())
			cb.Rule.Check("kubectl-delete-raw-filename", func(_ context.Context, cfg *deleteConfig, _ cli.Dependencies) (bool, error) {
				if cfg.Raw == "" || len(cfg.Filename.Filenames()) == 0 {
					return true, nil
				}
				filename := cfg.Filename.Filenames()[0]
				return !strings.HasPrefix(filename, "http://") && !strings.HasPrefix(filename, "https://"), nil
			}).
				DependsOn(raw.OptionalDependency()).
				DependsOn(filenameValue.OptionalDependency())
			cb.Values.Derive(&cb.Config.GracePeriod, cli.Value.ComputedBy(func(_ context.Context, _ *deleteConfig, dependencies cli.Dependencies) (int, error) {
				gracePeriod, _ := dependencies.Get(requestedGrace)
				forceDelete, _ := dependencies.Get(force)
				deleteNow, _ := dependencies.Get(now)
				if deleteNow {
					return 1, nil
				}
				if gracePeriod == 0 && !forceDelete {
					return 1, nil
				}
				if forceDelete && gracePeriod < 0 {
					return 0, nil
				}
				return gracePeriod, nil
			}).From(requestedGrace, force.OptionalDependency(), now.OptionalDependency()))

			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			cb.Values.ResolveWith(timeoutFlag, cli.Value.KeepLast())
			timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")

			waitFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Wait))
			cb.Values.ResolveWith(waitFlag, cli.Value.KeepLast())
			waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			cb.Values.Default(waitFlag, true)
			waitFlag.Long("wait").ImplicitValue(true)
		},
		Run: runDelete,
	}
	return cmd
}

package main

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/distribution/reference"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

type runAnnotation struct {
	Name  string
	Value string
}

type runCascade struct {
	Source     string
	Strategy   string
	Deprecated bool
}

var runCascadeLanguage = cli.ValueCodec[runCascade]{
	Name: "kubectl-cascade",
	Parse: func(value string) (runCascade, error) {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			strategy := "orphan"
			if boolValue {
				strategy = "background"
			}
			cascade := runCascade{
				Source:     value,
				Strategy:   strategy,
				Deprecated: true,
			}
			return cascade, nil
		}
		switch value {
		case "background", "foreground", "orphan":
			cascade := runCascade{
				Source:   value,
				Strategy: value,
			}
			return cascade, nil
		default:
			return runCascade{}, cli.Rejection.Message(
				"cascade",
				fmt.Sprintf(`invalid cascade value (%v). Must be "background", "foreground", or "orphan"`, value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
	},
	Format:   func(value runCascade) string { return value.Source },
	Snapshot: func(value runCascade) runCascade { return value },
}

type runConfig struct {
	Client                   clientConfig
	Streams                  genericiooptions.IOStreams
	Name                     string
	Arguments                []string
	AllowMissingTemplateKeys bool
	Annotations              []runAnnotation
	Attach                   bool
	Cascade                  runCascade
	Command                  bool
	DetachKeys               string
	DryRunStrategy           util.DryRunStrategy
	Env                      []v1.EnvVar
	Expose                   bool
	FieldManager             string
	Filename                 [][]string
	Force                    bool
	GracePeriod              int
	Image                    string
	ImagePullPolicy          v1.PullPolicy
	Kustomize                string
	Labels                   map[string]string
	LeaveStdinOpen           bool
	Output                   string
	OutputState              cli.OptionalValue[cli.ValueState]
	OverrideTypeInput        string
	OverrideType             util.OverrideType
	Overrides                string
	PodRunningTimeout        time.Duration
	Port                     *int
	Privileged               bool
	Quiet                    bool
	Record                   bool
	Recursive                bool
	Restart                  v1.RestartPolicy
	RestartPolicy            v1.RestartPolicy
	Rm                       bool
	SaveConfig               bool
	ShowManagedFields        bool
	Stdin                    bool
	Template                 string
	Timeout                  time.Duration
	TTY                      bool
	Wait                     bool
}

var runAnnotationLanguage = cli.ValueCodec[runAnnotation]{
	Name: "kubectl-annotation",
	Parse: func(value string) (runAnnotation, error) {
		name, annotationValue, found := strings.Cut(value, "=")
		if !found || name == "" {
			return runAnnotation{}, cli.Rejection.Message(
				"annotation",
				fmt.Sprintf("invalid annotation format: %s", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		annotation := runAnnotation{
			Name:  name,
			Value: annotationValue,
		}
		return annotation, nil
	},
	Format:   func(value runAnnotation) string { return value.Name + "=" + value.Value },
	Snapshot: func(value runAnnotation) runAnnotation { return value },
}

var runEnvironmentLanguage = cli.ValueCodec[v1.EnvVar]{
	Name: "kubectl-environment",
	Parse: func(value string) (v1.EnvVar, error) {
		name, environmentValue, found := strings.Cut(value, "=")
		if !found || name == "" || len(validation.IsEnvVarName(name)) != 0 {
			return v1.EnvVar{}, cli.Rejection.Message(
				"environment",
				fmt.Sprintf("invalid env: %s", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		environment := v1.EnvVar{
			Name:  name,
			Value: environmentValue,
		}
		return environment, nil
	},
	Format:   func(value v1.EnvVar) string { return value.Name + "=" + value.Value },
	Snapshot: func(value v1.EnvVar) v1.EnvVar { return value },
}

var runImageLanguage = cli.ValueCodec[string]{
	Name: "container-image",
	Parse: func(value string) (string, error) {
		if !reference.ReferenceRegexp.MatchString(value) {
			return "", cli.Rejection.Message(
				"container-image",
				fmt.Sprintf("invalid image name %q", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return value, nil
	},
	Format:   func(value string) string { return value },
	Snapshot: func(value string) string { return value },
}

var runRestartPolicyLanguage = cli.ValueCodec[v1.RestartPolicy]{
	Name: "restart-policy",
	Parse: func(value string) (v1.RestartPolicy, error) {
		policy := v1.RestartPolicy(value)
		switch policy {
		case v1.RestartPolicyAlways, v1.RestartPolicyOnFailure, v1.RestartPolicyNever, "":
			return policy, nil
		default:
			return "", cli.Rejection.Message(
				"restart-policy",
				fmt.Sprintf("invalid restart policy: %s, valid values are: %s, %s, %s", value, v1.RestartPolicyAlways, v1.RestartPolicyOnFailure, v1.RestartPolicyNever),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
	},
	Format:   func(value v1.RestartPolicy) string { return string(value) },
	Snapshot: func(value v1.RestartPolicy) v1.RestartPolicy { return value },
}

var runImagePullPolicyLanguage = cli.ValueCodec[v1.PullPolicy]{
	Name: "image-pull-policy",
	Parse: func(value string) (v1.PullPolicy, error) {
		policy := v1.PullPolicy(value)
		switch policy {
		case v1.PullAlways, v1.PullIfNotPresent, v1.PullNever, "":
			return policy, nil
		default:
			return "", cli.Rejection.Message(
				"image-pull-policy",
				fmt.Sprintf("invalid image pull policy: %s", value),
				cli.Rejection.At(0, 0, len(value)),
			)
		}
	},
	Format:   func(value v1.PullPolicy) string { return string(value) },
	Snapshot: func(value v1.PullPolicy) v1.PullPolicy { return value },
}

var runLabelsLanguage = cli.ValueCodec[map[string]string]{
	Name: "kubectl-labels",
	Parse: func(value string) (map[string]string, error) {
		if value == "" {
			return nil, nil
		}
		labels := make(map[string]string)
		for _, item := range strings.Split(value, ",") {
			parts := strings.Split(item, "=")
			if len(parts) != 2 || parts[0] == "" {
				return nil, cli.Rejection.Message(
					"labels",
					fmt.Sprintf("unexpected label spec: %s", item),
					cli.Rejection.At(0, 0, len(value)),
				)
			}
			labels[parts[0]] = parts[1]
		}
		return labels, nil
	},
	Format: func(value map[string]string) string {
		if len(value) == 0 {
			return ""
		}
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, key+"="+value[key])
		}
		return strings.Join(parts, ",")
	},
	Snapshot: func(value map[string]string) map[string]string { return maps.Clone(value) },
}

var runPortLanguage = cli.ValueCodec[*int]{
	Name: "port",
	Parse: func(value string) (*int, error) {
		port, err := strconv.Atoi(value)
		if err != nil {
			return nil, cli.Rejection.Message(
				"port",
				"--port must be a number",
				cli.Rejection.At(0, 0, len(value)),
			)
		}
		return &port, nil
	},
	Format: func(value *int) string {
		if value == nil {
			return ""
		}
		return strconv.Itoa(*value)
	},
	Snapshot: func(value *int) *int {
		if value == nil {
			return nil
		}
		port := *value
		return &port
	},
}

func runCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[runConfig] {
	cmd := cli.Command[runConfig]{
		Name: "run",
		Description: cli.Description{
			Short: "Run a particular image on the cluster",
			Long:  "Create and run a particular image in a pod.",
		},
		Examples: []cli.Example{
			{
				Description: "Start a nginx pod",
				CommandLine: "kubectl run nginx --image=nginx",
			},
			{
				Description: "Start a hazelcast pod and let the container expose port 5701",
				CommandLine: "kubectl run hazelcast --image=hazelcast/hazelcast --port=5701",
			},
			{
				Description: "Start a hazelcast pod and set environment variables \"DNS_DOMAIN=cluster\" and \"POD_NAMESPACE=default\" in the container",
				CommandLine: "kubectl run hazelcast --image=hazelcast/hazelcast --env=\"DNS_DOMAIN=cluster\" --env=\"POD_NAMESPACE=default\"",
			},
			{
				Description: "Start a hazelcast pod and set labels \"app=hazelcast\" and \"env=prod\" in the container",
				CommandLine: "kubectl run hazelcast --image=hazelcast/hazelcast --labels=\"app=hazelcast,env=prod\"",
			},
			{
				Description: "Dry run; print the corresponding API objects without creating them",
				CommandLine: "kubectl run nginx --image=nginx --dry-run=client",
			},
			{
				Description: "Start a nginx pod, but overload the spec with a partial set of values parsed from JSON",
				CommandLine: "kubectl run nginx --image=nginx --overrides='{ \"apiVersion\": \"v1\", \"spec\": { ... } }'",
			},
			{
				Description: "Start a busybox pod and keep it in the foreground, don't restart it if it exits",
				CommandLine: "kubectl run -i -t busybox --image=busybox --restart=Never",
			},
			{
				Description: "Start the nginx pod using the default command, but use custom arguments (arg1 .. argN) for that command",
				CommandLine: "kubectl run nginx --image=nginx -- <arg1> <arg2> ... <argN>",
			},
			{
				Description: "Start the nginx pod using a different command and custom arguments",
				CommandLine: "kubectl run nginx --image=nginx --command -- <cmd> <arg1> ... <argN>",
			},
		},
		Declare: func(cb *cli.CommandBuilder[runConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			declarePrintFlags(
				cb,
				&cb.Config.Output,
				&cb.Config.OutputState,
				&cb.Config.Template,
				&cb.Config.AllowMissingTemplateKeys,
				&cb.Config.ShowManagedFields,
			)
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[runConfig]) {
				form.String(&form.Config.Name, "NAME")
				form.PassThrough(arguments, "ARG", cli.Count.ZeroOrMore())
			})
			annotationsValue := cb.Values.Sequence(&cb.Config.Annotations, runAnnotationLanguage)
			annotationsFlag := cb.Flags.Sequence(annotationsValue)
			annotationsFlag.Description("Annotations to apply to the pod.")
			annotationsFlag.Long("annotations")
			attachFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Attach))
			cb.Values.ResolveWith(attachFlag, cli.Value.KeepLast())
			attachFlag.Description("If true, wait for the Pod to start running, and then attach to the Pod as if 'kubectl attach ...' were called.  Default false, unless '-i/--stdin' is set, in which case the default is true. With '--restart=Never' the exit code of the container process is returned.")
			attachFlag.Long("attach").ImplicitValue(true)
			cascadeFlag := cb.Flags.Value(&cb.Config.Cascade, runCascadeLanguage)
			cb.Values.ResolveWith(cascadeFlag, cli.Value.KeepLast())
			cascadeFlag.Description("Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents (e.g. Pods created by a ReplicationController). Defaults to background.")
			cb.Values.Default(cascadeFlag, runCascade{Source: "background", Strategy: "background"})
			cascadeFlag.Long("cascade").ImplicitValue(runCascade{Source: "background", Strategy: "background"})
			commandFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Command))
			cb.Values.ResolveWith(commandFlag, cli.Value.KeepLast())
			commandFlag.Description("If true and extra arguments are present, use them as the 'command' field in the container, rather than the 'args' field which is the default.")
			commandFlag.Long("command").ImplicitValue(true)
			detachKeysFlag := cb.Flags.String(&cb.Config.DetachKeys)
			cb.Values.ResolveWith(detachKeysFlag, cli.Value.KeepLast())
			detachKeysFlag.Description("Override the key sequence for detaching a container.")
			cb.Values.Default(detachKeysFlag, "ctrl-p,ctrl-q")
			detachKeysFlag.Long("detach-keys")
			dryRunFlag := cb.Flags.Value(&cb.Config.DryRunStrategy, dryRunValueLanguage)
			cb.Values.ResolveWith(dryRunFlag, cli.Value.KeepLast())
			dryRunFlag.Description("Must be \"none\", \"server\", or \"client\". If client strategy, only print the object that would be sent, without sending it. If server strategy, submit server-side request without persisting the resource.")
			cb.Values.Default(dryRunFlag, util.DryRunNone)
			dryRunFlag.Long("dry-run").ImplicitValue(util.DryRunClient)
			envValue := cb.Values.Sequence(&cb.Config.Env, runEnvironmentLanguage)
			envFlag := cb.Flags.Sequence(envValue)
			envFlag.Description("Environment variables to set in the container.")
			envFlag.Long("env")
			exposeFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Expose))
			cb.Values.ResolveWith(exposeFlag, cli.Value.KeepLast())
			exposeFlag.Description("If true, create a ClusterIP service associated with the pod.  Requires `--port`.")
			exposeFlag.Long("expose").ImplicitValue(true)
			fieldManagerFlag := cb.Flags.String(&cb.Config.FieldManager)
			cb.Values.ResolveWith(fieldManagerFlag, cli.Value.KeepLast())
			fieldManagerFlag.Description("Name of the manager used to track field ownership.")
			cb.Values.Default(fieldManagerFlag, "kubectl-run")
			fieldManagerFlag.Long("field-manager")
			filenameValue := cb.Values.Sequence(&cb.Config.Filename, filenameListLanguage)
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.Description("to use to replace the resource.")
			filenameFlag.Hidden().Release(preexistingDeprecation("it is ignored by kubectl run and will be removed in a future release"))
			filenameFlag.Long("filename")
			filenameFlag.Short('f').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			forceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Force))
			cb.Values.ResolveWith(forceFlag, cli.Value.KeepLast())
			forceFlag.Description("If true, immediately remove resources from API and bypass graceful deletion. Note that immediate deletion of some resources may result in inconsistency or data loss and requires confirmation.")
			forceFlag.Long("force").ImplicitValue(true)
			gracePeriodFlag := cb.Flags.Int(&cb.Config.GracePeriod)
			cb.Values.ResolveWith(gracePeriodFlag, cli.Value.KeepLast())
			gracePeriodFlag.Description("Period of time in seconds given to the resource to terminate gracefully. Ignored if negative. Set to 1 for immediate shutdown. Can only be set to 0 when --force is true (force deletion).")
			cb.Values.Default(gracePeriodFlag, -1)
			gracePeriodFlag.Long("grace-period")
			imageFlag := cb.Flags.Value(&cb.Config.Image, runImageLanguage)
			cb.Values.ResolveWith(imageFlag, cli.Value.KeepLast())
			imageFlag.Description("The image for the container to run.")
			imageFlag.Long("image")
			imagePullPolicyFlag := cb.Flags.Value(&cb.Config.ImagePullPolicy, runImagePullPolicyLanguage)
			cb.Values.ResolveWith(imagePullPolicyFlag, cli.Value.KeepLast())
			imagePullPolicyFlag.Description("The image pull policy for the container.  If left empty, this value will not be specified by the client and defaulted by the server.")
			imagePullPolicyFlag.Long("image-pull-policy")
			kustomizeFlag := cb.Flags.String(&cb.Config.Kustomize)
			cb.Values.ResolveWith(kustomizeFlag, cli.Value.KeepLast())
			kustomizeFlag.Description("Process a kustomization directory. This flag can't be used together with -f or -R.")
			kustomizeFlag.Long("kustomize")
			kustomizeFlag.Short('k').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			labelsFlag := cb.Flags.Value(&cb.Config.Labels, runLabelsLanguage)
			cb.Values.ResolveWith(labelsFlag, cli.Value.KeepLast())
			labelsFlag.Description("Comma separated labels to apply to the pod. Will override previous values.")
			labelsFlag.Long("labels")
			labelsFlag.Short('l').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			leaveStdinOpenFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.LeaveStdinOpen))
			cb.Values.ResolveWith(leaveStdinOpenFlag, cli.Value.KeepLast())
			leaveStdinOpenFlag.Description("If the pod is started in interactive mode or with stdin, leave stdin open after the first attach completes. By default, stdin will be closed after the first attach completes.")
			leaveStdinOpenFlag.Long("leave-stdin-open").ImplicitValue(true)
			overrideTypeFlag := cb.Flags.String(&cb.Config.OverrideTypeInput)
			cb.Values.ResolveWith(overrideTypeFlag, cli.Value.KeepLast())
			overrideTypeFlag.Description("The method used to override the generated object: json, merge, or strategic.")
			cb.Values.Default(overrideTypeFlag, string(util.DefaultOverrideType))
			overrideTypeFlag.Long("override-type")
			overridesFlag := cb.Flags.String(&cb.Config.Overrides)
			cb.Values.ResolveWith(overridesFlag, cli.Value.KeepLast())
			overridesFlag.Description("An inline JSON override for the generated object. If this is non-empty, it is used to override the generated object. Requires that the object supply a valid apiVersion field.")
			overridesFlag.Long("overrides")
			cb.Values.Derive(
				&cb.Config.OverrideType,
				cli.Value.ComputedBy(func(_ context.Context, cfg *runConfig, _ cli.Dependencies) (util.OverrideType, error) {
					if cfg.Overrides == "" || cfg.OverrideTypeInput == "" {
						return util.DefaultOverrideType, nil
					}
					return util.OverrideType(cfg.OverrideTypeInput), nil
				}).From(overrideTypeFlag, overridesFlag.OptionalDependency()),
			)
			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			cb.Values.ResolveWith(podRunningTimeoutFlag, cli.Value.KeepLast())
			podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, time.Minute)
			podRunningTimeoutFlag.Long("pod-running-timeout")
			portFlag := cb.Flags.Value(&cb.Config.Port, runPortLanguage)
			cb.Values.ResolveWith(portFlag, cli.Value.KeepLast())
			portFlag.Description("The port that this container exposes.")
			portFlag.Long("port")
			privilegedFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Privileged))
			cb.Values.ResolveWith(privilegedFlag, cli.Value.KeepLast())
			privilegedFlag.Description("If true, run the container in privileged mode.")
			privilegedFlag.Long("privileged").ImplicitValue(true)
			quietFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Quiet))
			cb.Values.ResolveWith(quietFlag, cli.Value.KeepLast())
			quietFlag.Description("If true, suppress prompt messages.")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			recordFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Record))
			cb.Values.ResolveWith(recordFlag, cli.Value.KeepLast())
			recordFlag.Description("Record current kubectl command in the resource annotation. If set to false, do not record the command. If set to true, record the command. If not set, default to updating the existing annotation value only if one already exists.")
			recordFlag.Hidden().Release(preexistingDeprecation("--record will be removed in the future"))
			recordFlag.Long("record").ImplicitValue(true)
			recursiveFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Recursive))
			cb.Values.ResolveWith(recursiveFlag, cli.Value.KeepLast())
			recursiveFlag.Description("Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory.")
			recursiveFlag.Long("recursive").ImplicitValue(true)
			recursiveFlag.Short('R').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			restartFlag := cb.Flags.Value(&cb.Config.Restart, runRestartPolicyLanguage)
			cb.Values.ResolveWith(restartFlag, cli.Value.KeepLast())
			restartFlag.Description("The restart policy for this Pod.  Legal values [Always, OnFailure, Never].")
			cb.Values.Default(restartFlag, v1.RestartPolicyAlways)
			restartFlag.Long("restart")
			rmFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Rm))
			cb.Values.ResolveWith(rmFlag, cli.Value.KeepLast())
			rmFlag.Description("If true, delete the pod after it exits.  Only valid when attaching to the container, e.g. with '--attach' or with '-i/--stdin'.")
			rmFlag.Long("rm").ImplicitValue(true)
			saveConfigFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SaveConfig))
			cb.Values.ResolveWith(saveConfigFlag, cli.Value.KeepLast())
			saveConfigFlag.Description("If true, the configuration of current object will be saved in its annotation. Otherwise, the annotation will be unchanged. This flag is useful when you want to perform kubectl apply on this object in the future.")
			saveConfigFlag.Long("save-config").ImplicitValue(true)
			stdinFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Stdin))
			cb.Values.ResolveWith(stdinFlag, cli.Value.KeepLast())
			stdinFlag.Description("Keep stdin open on the container in the pod, even if nothing is attached.")
			stdinFlag.Long("stdin").ImplicitValue(true)
			stdinFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			timeoutFlag := cb.Flags.Value(&cb.Config.Timeout, durationLanguage())
			cb.Values.ResolveWith(timeoutFlag, cli.Value.KeepLast())
			timeoutFlag.Description("The length of time to wait before giving up on a delete, zero means determine a timeout from the size of the object")
			timeoutFlag.Long("timeout")
			ttyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.TTY))
			cb.Values.ResolveWith(ttyFlag, cli.Value.KeepLast())
			ttyFlag.Description("Allocate a TTY for the container in the pod.")
			ttyFlag.Long("tty").ImplicitValue(true)
			ttyFlag.Short('t').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			waitFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Wait))
			cb.Values.ResolveWith(waitFlag, cli.Value.KeepLast())
			waitFlag.Description("If true, wait for resources to be gone before returning. This waits for finalizers.")
			waitFlag.Long("wait").ImplicitValue(true)
			cb.Values.ComputedDefault(
				attachFlag.Destination(),
				cli.Value.ComputedBy(func(_ context.Context, cfg *runConfig, _ cli.Dependencies) (bool, error) {
					return cfg.Stdin, nil
				}).From(stdinFlag.OptionalDependency()),
			)
			cb.Values.Derive(
				&cb.Config.RestartPolicy,
				cli.Value.ComputedBy(func(_ context.Context, cfg *runConfig, _ cli.Dependencies) (v1.RestartPolicy, error) {
					if cfg.Restart == "" {
						if cfg.Stdin {
							return v1.RestartPolicyOnFailure, nil
						}
						return v1.RestartPolicyAlways, nil
					}
					return cfg.Restart, nil
				}).From(restartFlag, stdinFlag.OptionalDependency()),
			)
			cb.Rule.When(cb.When.NotEqual(overridesFlag.Destination(), "")).In(
				overrideTypeFlag.Destination(),
				"",
				string(util.OverrideTypeJSON),
				string(util.OverrideTypeMerge),
				string(util.OverrideTypeStrategic),
			)
			cb.Rule.Require(imageFlag.Destination())
			cb.Rule.When(cb.When.Equal(ttyFlag.Destination(), true)).Check("kubectl-run-tty-stdin", func(_ context.Context, cfg *runConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Stdin, nil
			}).DependsOn(stdinFlag.OptionalDependency())
			cb.Rule.When(cb.When.Equal(exposeFlag.Destination(), true)).Require(portFlag.Destination())
			cb.Rule.When(cb.When.Equal(rmFlag.Destination(), true)).Equal(attachFlag.Destination(), true)
			cb.Rule.When(cb.When.Equal(attachFlag.Destination(), true)).Equal(dryRunFlag.Destination(), util.DryRunNone)
			cb.Rule.GreaterThan(podRunningTimeoutFlag.Destination(), time.Duration(0))
		},
		Run: runRun,
	}
	return cmd
}

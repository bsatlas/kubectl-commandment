package main

import (
	"context"
	"fmt"

	"github.com/abstractionsinc/commandment/cli"
	"github.com/distribution/reference"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/cmd/debug"
)

type debugConfig struct {
	Client              clientConfig
	Streams             genericiooptions.IOStreams
	TargetNames         []string
	Arguments           []string
	ArgumentsOnly       bool
	Attach              bool
	Container           string
	CopyTo              string
	Custom              string
	Env                 map[string]string
	Filename            []string
	Image               string
	ImagePullPolicy     v1.PullPolicy
	KeepAnnotations     bool
	KeepInitContainers  bool
	KeepLabels          bool
	KeepLiveness        bool
	KeepReadiness       bool
	KeepStartup         bool
	Profile             string
	Quiet               bool
	Replace             bool
	SameNode            bool
	SetImage            map[string]string
	ShareProcesses      bool
	ShareProcessesState cli.OptionalValue[cli.ValueState]
	Stdin               bool
	Target              string
	TTY                 bool
}

var debugOptionalImageLanguage = cli.ValueCodec[string]{
	Name: "debug-container-image",
	Parse: func(value string) (string, error) {
		if value != "" && !reference.ReferenceRegexp.MatchString(value) {
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

var debugImageLanguage = cli.ValueCodec[string]{
	Name: "debug-set-image",
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

var debugImagePullPolicyLanguage = cli.ValueCodec[v1.PullPolicy]{
	Name: "debug-image-pull-policy",
	Parse: func(value string) (v1.PullPolicy, error) {
		policy := v1.PullPolicy(value)
		switch policy {
		case "", v1.PullAlways, v1.PullIfNotPresent, v1.PullNever:
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

func debugProfileLanguage() cli.ValueDomain[string] {
	language := cli.Value.Domain(
		cli.Value.Assign[string](),
		func(values *cli.ValueDomainBuilder[string]) {
			values.Value(debug.ProfileLegacy, debug.ProfileLegacy).Release(preexistingDeprecation(
				`--profile=legacy is deprecated and planned to be removed in v1.39; use another profile such as --profile=general`,
			))
			values.Value(debug.ProfileGeneral, debug.ProfileGeneral)
			values.Value(debug.ProfileBaseline, debug.ProfileBaseline)
			values.Value(debug.ProfileRestricted, debug.ProfileRestricted)
			values.Value(debug.ProfileNetadmin, debug.ProfileNetadmin)
			values.Value(debug.ProfileSysadmin, debug.ProfileSysadmin)
		},
	)
	return language
}

func debugCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[debugConfig] {
	cmd := cli.Command[debugConfig]{
		Name: "debug",
		Description: cli.Description{
			Short: "Create debugging sessions for troubleshooting workloads and nodes",
			Long:  "Debug cluster resources using interactive debugging containers.\n\n 'debug' provides automation for common debugging tasks for cluster objects identified by resource and name. Pods will be used by default if no resource is specified.\n\n The action taken by 'debug' varies depending on what resource is specified. Supported actions include:\n\n  *  Workload: Create a copy of an existing pod with certain attributes changed, for example changing the image tag to a new version.\n  *  Workload: Add an ephemeral container to an already running pod, for example to add debugging utilities without restarting the pod.\n  *  Node: Create a new pod that runs in the node's host namespaces and can access the node's filesystem.\n\n Note: When a non-root user is configured for the entire target Pod, some capabilities granted by debug profile may not work.",
		},
		Examples: []cli.Example{
			{
				Description: "Create an interactive debugging session in pod mypod and immediately attach to it.",
				CommandLine: "kubectl debug mypod -it --image=busybox",
			},
			{
				Description: "Create an interactive debugging session for the pod in the file pod.yaml and immediately attach to it.\n(requires the EphemeralContainers feature to be enabled in the cluster)",
				CommandLine: "kubectl debug -f pod.yaml -it --image=busybox",
			},
			{
				Description: "Create a debug container named debugger using a custom automated debugging image.",
				CommandLine: "kubectl debug --image=myproj/debug-tools -c debugger mypod",
			},
			{
				Description: "Create a copy of mypod adding a debug container and attach to it",
				CommandLine: "kubectl debug mypod -it --image=busybox --copy-to=my-debugger",
			},
			{
				Description: "Create a copy of mypod changing the command of mycontainer",
				CommandLine: "kubectl debug mypod -it --copy-to=my-debugger --container=mycontainer -- sh",
			},
			{
				Description: "Create a copy of mypod changing all container images to busybox",
				CommandLine: "kubectl debug mypod --copy-to=my-debugger --set-image=*=busybox",
			},
			{
				Description: "Create a copy of mypod adding a debug container and changing container images",
				CommandLine: "kubectl debug mypod -it --copy-to=my-debugger --image=debian --set-image=app=app:debug,sidecar=sidecar:debug",
			},
			{
				Description: "Create an interactive debugging session on a node and immediately attach to it.\nThe container will run in the host namespaces and the host's filesystem will be mounted at /host",
				CommandLine: "kubectl debug node/mynode -it --image=busybox",
			},
		},
		Declare: func(cb *cli.CommandBuilder[debugConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			targets := cb.Values.Sequence(&cb.Config.TargetNames, cli.Value.String())
			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[debugConfig]) {
				form.Sequence(targets, "TARGET", cli.Count.ZeroOrMore()).
					Greedy().
					CompleteBy(resourceArgumentCompleter(
						func(cfg *debugConfig) (clientConfig, []string) { return cfg.Client, cfg.TargetNames },
						nil,
						resourceArgumentRepeated,
					)).
					From(client)
				form.PassThrough(arguments, "COMMAND", cli.Count.ZeroOrMore())
			})
			argumentsOnlyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ArgumentsOnly))
			cb.Values.ResolveWith(argumentsOnlyFlag, cli.Value.KeepLast())
			argumentsOnlyFlag.Description("If specified, everything after -- will be passed to the new container as Args instead of Command.")
			argumentsOnlyFlag.Long("arguments-only").ImplicitValue(true)
			attachFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Attach))
			cb.Values.ResolveWith(attachFlag, cli.Value.KeepLast())
			attachFlag.Description("If true, wait for the container to start running, and then attach as if 'kubectl attach ...' were called.  Default false, unless '-i/--stdin' is set, in which case the default is true.")
			attachFlag.Long("attach").ImplicitValue(true)
			containerFlag := cb.Flags.String(&cb.Config.Container)
			cb.Values.ResolveWith(containerFlag, cli.Value.KeepLast())
			containerFlag.Description("Container name to use for debug container.")
			containerFlag.Long("container")
			containerFlag.Short('c')
			copyToFlag := cb.Flags.String(&cb.Config.CopyTo)
			cb.Values.ResolveWith(copyToFlag, cli.Value.KeepLast())
			copyToFlag.Description("Create a copy of the target Pod with this name.")
			copyToFlag.Long("copy-to")
			customFlag := cb.Flags.String(&cb.Config.Custom)
			cb.Values.ResolveWith(customFlag, cli.Value.KeepLast())
			customFlag.Description("Path to a JSON or YAML file containing a partial container spec to customize built-in debug profiles.")
			customFlag.Long("custom")
			envValue := cb.Values.Keyed(&cb.Config.Env, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			cb.Values.ResolveWith(envValue, cli.Value.KeepLast())
			envFlag := cb.Flags.Keyed(envValue)
			envFlag.DelimitedBy(",")
			envFlag.Description("Environment variables to set in the container.")
			envFlag.Long("env")
			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("identifying the resource to debug")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')
			imageFlag := cb.Flags.Value(&cb.Config.Image, debugOptionalImageLanguage)
			cb.Values.ResolveWith(imageFlag, cli.Value.KeepLast())
			imageFlag.Description("Container image to use for debug container.")
			imageFlag.Long("image")
			imagePullPolicyFlag := cb.Flags.Value(&cb.Config.ImagePullPolicy, debugImagePullPolicyLanguage)
			cb.Values.ResolveWith(imagePullPolicyFlag, cli.Value.KeepLast())
			imagePullPolicyFlag.Description("The image pull policy for the container. If left empty, this value will not be specified by the client and defaulted by the server.")
			imagePullPolicyFlag.Long("image-pull-policy")
			keepAnnotationsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepAnnotations))
			cb.Values.ResolveWith(keepAnnotationsFlag, cli.Value.KeepLast())
			keepAnnotationsFlag.Description("If true, keep the original pod annotations.(This flag only works when used with '--copy-to')")
			keepAnnotationsFlag.Long("keep-annotations").ImplicitValue(true)
			keepInitContainersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepInitContainers))
			cb.Values.ResolveWith(keepInitContainersFlag, cli.Value.KeepLast())
			keepInitContainersFlag.Description("Run the init containers for the pod. Defaults to true.(This flag only works when used with '--copy-to')")
			cb.Values.Default(keepInitContainersFlag, true)
			keepInitContainersFlag.Long("keep-init-containers").ImplicitValue(true)
			keepLabelsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepLabels))
			cb.Values.ResolveWith(keepLabelsFlag, cli.Value.KeepLast())
			keepLabelsFlag.Description("If true, keep the original pod labels.(This flag only works when used with '--copy-to')")
			keepLabelsFlag.Long("keep-labels").ImplicitValue(true)
			keepLivenessFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepLiveness))
			cb.Values.ResolveWith(keepLivenessFlag, cli.Value.KeepLast())
			keepLivenessFlag.Description("If true, keep the original pod liveness probes.(This flag only works when used with '--copy-to')")
			keepLivenessFlag.Long("keep-liveness").ImplicitValue(true)
			keepReadinessFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepReadiness))
			cb.Values.ResolveWith(keepReadinessFlag, cli.Value.KeepLast())
			keepReadinessFlag.Description("If true, keep the original pod readiness probes.(This flag only works when used with '--copy-to')")
			keepReadinessFlag.Long("keep-readiness").ImplicitValue(true)
			keepStartupFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepStartup))
			cb.Values.ResolveWith(keepStartupFlag, cli.Value.KeepLast())
			keepStartupFlag.Description("If true, keep the original startup probes.(This flag only works when used with '--copy-to')")
			keepStartupFlag.Long("keep-startup").ImplicitValue(true)
			profileFlag := cb.Flags.Value(&cb.Config.Profile, debugProfileLanguage())
			cb.Values.ResolveWith(profileFlag, cli.Value.KeepLast())
			profileFlag.Description("Options are \"general\", \"baseline\", \"restricted\", \"netadmin\" or \"sysadmin\". Defaults to \"general\"")
			cb.Values.Default(profileFlag, "general")
			profileFlag.Long("profile")
			quietFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Quiet))
			cb.Values.ResolveWith(quietFlag, cli.Value.KeepLast())
			quietFlag.Description("If true, suppress informational messages.")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			replaceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Replace))
			cb.Values.ResolveWith(replaceFlag, cli.Value.KeepLast())
			replaceFlag.Description("When used with '--copy-to', delete the original Pod.")
			replaceFlag.Long("replace").ImplicitValue(true)
			sameNodeFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SameNode))
			cb.Values.ResolveWith(sameNodeFlag, cli.Value.KeepLast())
			sameNodeFlag.Description("When used with '--copy-to', schedule the copy of target Pod on the same node.")
			sameNodeFlag.Long("same-node").ImplicitValue(true)
			setImageValue := cb.Values.Keyed(&cb.Config.SetImage, cli.Value.Keyed(cli.Value.String(), debugImageLanguage))
			cb.Values.ResolveWith(setImageValue, cli.Value.KeepLast())
			setImageFlag := cb.Flags.Keyed(setImageValue)
			setImageFlag.DelimitedBy(",")
			setImageFlag.Description("When used with '--copy-to', a list of name=image pairs for changing container images, similar to how 'kubectl set image' works.")
			setImageFlag.Long("set-image")
			shareProcessesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShareProcesses))
			cb.Values.ResolveWith(shareProcessesFlag, cli.Value.KeepLast())
			shareProcessesFlag.Description("When used with '--copy-to', enable process namespace sharing in the copy.")
			cb.Values.Default(shareProcessesFlag, true)
			shareProcessesFlag.Long("share-processes").ImplicitValue(true)
			cb.Access.ValueState(shareProcessesFlag, &cb.Config.ShareProcessesState)
			stdinFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Stdin))
			cb.Values.ResolveWith(stdinFlag, cli.Value.KeepLast())
			stdinFlag.Description("Keep stdin open on the container(s) in the pod, even if nothing is attached.")
			stdinFlag.Long("stdin").ImplicitValue(true)
			stdinFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			targetFlag := cb.Flags.String(&cb.Config.Target)
			cb.Values.ResolveWith(targetFlag, cli.Value.KeepLast())
			targetFlag.Description("When using an ephemeral container, target processes in this container name.")
			targetFlag.Long("target")
			ttyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.TTY))
			cb.Values.ResolveWith(ttyFlag, cli.Value.KeepLast())
			ttyFlag.Description("Allocate a TTY for the debugging container.")
			ttyFlag.Long("tty").ImplicitValue(true)
			ttyFlag.Short('t').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			cb.Values.ComputedDefault(
				attachFlag.Destination(),
				cli.Value.ComputedBy(func(_ context.Context, cfg *debugConfig, _ cli.Dependencies) (bool, error) {
					return cfg.Stdin, nil
				}).From(stdinFlag.OptionalDependency()),
			)
			copying := cb.When.NotEqual(copyToFlag.Destination(), "")
			attachExplicit := cb.Rule.All(
				cb.Rule.ContributedBy(attachFlag, cli.ValueProducerKindFlag),
				cb.When.Equal(attachFlag.Destination(), true),
			)
			cb.Rule.When(attachExplicit).Check("kubectl-debug-attach-target", func(_ context.Context, cfg *debugConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Image != "" || cfg.Container != "", nil
			}).
				DependsOn(imageFlag.OptionalDependency()).
				DependsOn(containerFlag.OptionalDependency())
			cb.Rule.When(copying).Check("kubectl-debug-copy-input", func(_ context.Context, cfg *debugConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Image != "" || len(cfg.SetImage) != 0 || len(cfg.Arguments) != 0, nil
			}).
				DependsOn(imageFlag.OptionalDependency()).
				DependsOn(setImageValue.OptionalDependency()).
				DependsOn(arguments.OptionalDependency())
			cb.Rule.When(copying).Check("kubectl-debug-copy-arguments-target", func(_ context.Context, cfg *debugConfig, _ cli.Dependencies) (bool, error) {
				return len(cfg.Arguments) == 0 || cfg.Container != "" || cfg.Image != "", nil
			}).
				DependsOn(arguments.OptionalDependency()).
				DependsOn(containerFlag.OptionalDependency()).
				DependsOn(imageFlag.OptionalDependency())
			cb.Rule.Unless(copying).Forbid(replaceFlag.Destination())
			cb.Rule.Unless(copying).Forbid(sameNodeFlag.Destination())
			cb.Rule.Unless(copying).Forbid(setImageValue)
			cb.Rule.Unless(copying).Require(imageFlag.Destination())
			cb.Rule.Unless(copying).NotEmpty(imageFlag.Destination())
			cb.Rule.AtLeastOne(targets, filenameValue)
			cb.Rule.When(copying).Equal(targetFlag.Destination(), "")
			cb.Rule.When(cb.When.Equal(ttyFlag.Destination(), true)).Check("kubectl-debug-tty-stdin", func(_ context.Context, cfg *debugConfig, _ cli.Dependencies) (bool, error) {
				return cfg.Stdin, nil
			}).DependsOn(stdinFlag.OptionalDependency())
		},
		Run: runDebug,
	}
	return cmd
}

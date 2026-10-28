package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type debugConfig struct {
	Client              clientConfig
	Streams             genericiooptions.IOStreams
	TargetNames         []string
	Arguments           []string
	ArgumentsOnly       bool
	Attach              bool
	AttachState         cli.OptionalValue[cli.ValueState]
	Container           string
	CopyTo              string
	Custom              string
	Env                 map[string]string
	Filename            []string
	Image               string
	ImagePullPolicy     string
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
			argumentsOnlyFlag = argumentsOnlyFlag.Description("If specified, everything after -- will be passed to the new container as Args instead of Command.")
			argumentsOnlyFlag.Long("arguments-only").ImplicitValue(true)

			attachFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Attach))
			attachFlag = attachFlag.Description("If true, wait for the container to start running, and then attach as if 'kubectl attach ...' were called.  Default false, unless '-i/--stdin' is set, in which case the default is true.")
			attachFlag.Long("attach").ImplicitValue(true)
			cb.Access.ValueState(attachFlag, &cb.Config.AttachState)

			containerFlag := cb.Flags.String(&cb.Config.Container)
			containerFlag = containerFlag.Description("Container name to use for debug container.")
			containerFlag.Long("container")
			containerFlag.Short('c')

			copyToFlag := cb.Flags.String(&cb.Config.CopyTo)
			copyToFlag = copyToFlag.Description("Create a copy of the target Pod with this name.")
			copyToFlag.Long("copy-to")

			customFlag := cb.Flags.String(&cb.Config.Custom)
			customFlag = customFlag.Description("Path to a JSON or YAML file containing a partial container spec to customize built-in debug profiles.")
			customFlag.Long("custom")

			envValue := cb.Values.Keyed(&cb.Config.Env, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			envFlag := cb.Flags.Keyed(envValue)
			envFlag = envFlag.DelimitedBy(",")
			envFlag = envFlag.Description("Environment variables to set in the container.")
			envFlag.Long("env")

			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag = filenameFlag.DelimitedBy(",")
			filenameFlag = filenameFlag.Description("identifying the resource to debug")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')

			imageFlag := cb.Flags.String(&cb.Config.Image)
			imageFlag = imageFlag.Description("Container image to use for debug container.")
			imageFlag.Long("image")

			imagePullPolicyFlag := cb.Flags.String(&cb.Config.ImagePullPolicy)
			imagePullPolicyFlag = imagePullPolicyFlag.Description("The image pull policy for the container. If left empty, this value will not be specified by the client and defaulted by the server.")
			imagePullPolicyFlag.Long("image-pull-policy")

			keepAnnotationsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepAnnotations))
			keepAnnotationsFlag = keepAnnotationsFlag.Description("If true, keep the original pod annotations.(This flag only works when used with '--copy-to')")
			keepAnnotationsFlag.Long("keep-annotations").ImplicitValue(true)

			keepInitContainersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepInitContainers))
			keepInitContainersFlag = keepInitContainersFlag.Description("Run the init containers for the pod. Defaults to true.(This flag only works when used with '--copy-to')")
			cb.Values.Default(keepInitContainersFlag, true)
			keepInitContainersFlag.Long("keep-init-containers").ImplicitValue(true)

			keepLabelsFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepLabels))
			keepLabelsFlag = keepLabelsFlag.Description("If true, keep the original pod labels.(This flag only works when used with '--copy-to')")
			keepLabelsFlag.Long("keep-labels").ImplicitValue(true)

			keepLivenessFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepLiveness))
			keepLivenessFlag = keepLivenessFlag.Description("If true, keep the original pod liveness probes.(This flag only works when used with '--copy-to')")
			keepLivenessFlag.Long("keep-liveness").ImplicitValue(true)

			keepReadinessFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepReadiness))
			keepReadinessFlag = keepReadinessFlag.Description("If true, keep the original pod readiness probes.(This flag only works when used with '--copy-to')")
			keepReadinessFlag.Long("keep-readiness").ImplicitValue(true)

			keepStartupFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.KeepStartup))
			keepStartupFlag = keepStartupFlag.Description("If true, keep the original startup probes.(This flag only works when used with '--copy-to')")
			keepStartupFlag.Long("keep-startup").ImplicitValue(true)

			profileFlag := cb.Flags.String(&cb.Config.Profile)
			profileFlag = profileFlag.Description("Options are \"general\", \"baseline\", \"restricted\", \"netadmin\" or \"sysadmin\". Defaults to \"general\"")
			cb.Values.Default(profileFlag, "general")
			profileFlag.Long("profile")

			quietFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Quiet))
			quietFlag = quietFlag.Description("If true, suppress informational messages.")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			replaceFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Replace))
			replaceFlag = replaceFlag.Description("When used with '--copy-to', delete the original Pod.")
			replaceFlag.Long("replace").ImplicitValue(true)

			sameNodeFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.SameNode))
			sameNodeFlag = sameNodeFlag.Description("When used with '--copy-to', schedule the copy of target Pod on the same node.")
			sameNodeFlag.Long("same-node").ImplicitValue(true)

			setImageValue := cb.Values.Keyed(&cb.Config.SetImage, cli.Value.Keyed(cli.Value.String(), cli.Value.String()))
			setImageFlag := cb.Flags.Keyed(setImageValue)
			setImageFlag = setImageFlag.DelimitedBy(",")
			setImageFlag = setImageFlag.Description("When used with '--copy-to', a list of name=image pairs for changing container images, similar to how 'kubectl set image' works.")
			setImageFlag.Long("set-image")

			shareProcessesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.ShareProcesses))
			shareProcessesFlag = shareProcessesFlag.Description("When used with '--copy-to', enable process namespace sharing in the copy.")
			cb.Values.Default(shareProcessesFlag, true)
			shareProcessesFlag.Long("share-processes").ImplicitValue(true)
			cb.Access.ValueState(shareProcessesFlag, &cb.Config.ShareProcessesState)

			stdinFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Stdin))
			stdinFlag = stdinFlag.Description("Keep stdin open on the container(s) in the pod, even if nothing is attached.")
			stdinFlag.Long("stdin").ImplicitValue(true)
			stdinFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			targetFlag := cb.Flags.String(&cb.Config.Target)
			targetFlag = targetFlag.Description("When using an ephemeral container, target processes in this container name.")
			targetFlag.Long("target")

			ttyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.TTY))
			ttyFlag = ttyFlag.Description("Allocate a TTY for the debugging container.")
			ttyFlag.Long("tty").ImplicitValue(true)
			ttyFlag.Short('t').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
		},
		Run: runDebug,
	}
	return cmd
}

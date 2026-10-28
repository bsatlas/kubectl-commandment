package main

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type attachConfig struct {
	Client            clientConfig
	Streams           genericiooptions.IOStreams
	Arguments         []string
	Container         string
	DetachKeys        string
	PodRunningTimeout time.Duration
	Quiet             bool
	Stdin             bool
	TTY               bool
}

func attachCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[attachConfig] {
	cmd := cli.Command[attachConfig]{
		Name: "attach",
		Description: cli.Description{
			Short: "Attach to a running container",
			Long:  "Attach to a process that is already running inside an existing container.",
		},
		Examples: []cli.Example{
			{
				Description: "Get output from running pod mypod; use the 'kubectl.kubernetes.io/default-container' annotation\nfor selecting the container to be attached or the first container in the pod will be chosen",
				CommandLine: "kubectl attach mypod",
			},
			{
				Description: "Get output from ruby-container from pod mypod",
				CommandLine: "kubectl attach mypod -c ruby-container",
			},
			{
				Description: "Switch to raw terminal mode; sends stdin to 'bash' in ruby-container from pod mypod\nand sends stdout/stderr from 'bash' back to the client",
				CommandLine: "kubectl attach mypod -c ruby-container -i -t",
			},
			{
				Description: "Get output from the first pod of a replica set named nginx",
				CommandLine: "kubectl attach rs/nginx",
			},
		},
		Declare: func(cb *cli.CommandBuilder[attachConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[attachConfig]) {
				form.Sequence(arguments, "resource", cli.Count.Between(1, 2)).
					CompleteBy(podResourceArgumentCompleter(
						func(cfg *attachConfig) (clientConfig, []string) { return cfg.Client, cfg.Arguments },
					)).
					From(client)
			})

			containerFlag := cb.Flags.String(&cb.Config.Container)
			containerFlag = containerFlag.Description("Container name. If omitted, use the kubectl.kubernetes.io/default-container annotation for selecting the container to be attached or the first container in the pod will be chosen")
			containerFlag.Long("container")
			containerFlag.Short('c')

			detachKeysFlag := cb.Flags.String(&cb.Config.DetachKeys)
			detachKeysFlag = detachKeysFlag.Description("Override the key sequence for detaching a container")
			cb.Values.Default(detachKeysFlag, "ctrl-p,ctrl-q")
			detachKeysFlag.Long("detach-keys")

			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			podRunningTimeoutFlag = podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, time.Minute)
			podRunningTimeoutFlag.Long("pod-running-timeout")

			quietFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Quiet))
			quietFlag = quietFlag.Description("Only print output from the remote session")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			stdinFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Stdin))
			stdinFlag = stdinFlag.Description("Pass stdin to the container")
			stdinFlag.Long("stdin").ImplicitValue(true)
			stdinFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			ttyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.TTY))
			ttyFlag = ttyFlag.Description("Stdin is a TTY")
			ttyFlag.Long("tty").ImplicitValue(true)
			ttyFlag.Short('t').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
		},
		Run: runAttach,
	}
	return cmd
}

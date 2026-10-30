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
			cb.Values.ResolveWith(containerFlag, cli.Value.KeepLast())
			containerFlag.Description("Container name. If omitted, use the kubectl.kubernetes.io/default-container annotation for selecting the container to be attached or the first container in the pod will be chosen")
			containerFlag.Long("container")
			containerFlag.Short('c').ValueForms(cli.FlagValueFormSeparate, cli.FlagValueFormJoined, cli.FlagValueFormAssigned)
			detachKeysFlag := cb.Flags.String(&cb.Config.DetachKeys)
			cb.Values.ResolveWith(detachKeysFlag, cli.Value.KeepLast())
			detachKeysFlag.Description("Override the key sequence for detaching a container")
			cb.Values.Default(detachKeysFlag, "ctrl-p,ctrl-q")
			detachKeysFlag.Long("detach-keys")
			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			cb.Values.ResolveWith(podRunningTimeoutFlag, cli.Value.KeepLast())
			podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, time.Minute)
			podRunningTimeoutFlag.Long("pod-running-timeout")
			cb.Rule.GreaterThan(podRunningTimeoutFlag.Destination(), time.Duration(0))
			quietFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Quiet))
			cb.Values.ResolveWith(quietFlag, cli.Value.KeepLast())
			quietFlag.Description("Only print output from the remote session")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			stdinFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Stdin))
			cb.Values.ResolveWith(stdinFlag, cli.Value.KeepLast())
			stdinFlag.Description("Pass stdin to the container")
			stdinFlag.Long("stdin").ImplicitValue(true)
			stdinFlag.Short('i').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
			ttyFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.TTY))
			cb.Values.ResolveWith(ttyFlag, cli.Value.KeepLast())
			ttyFlag.Description("Stdin is a TTY")
			ttyFlag.Long("tty").ImplicitValue(true)
			ttyFlag.Short('t').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)
		},
		Run: runAttach,
	}
	return cmd
}

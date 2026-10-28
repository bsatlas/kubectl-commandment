package kubectl

import (
	"time"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type execConfig struct {
	Client            clientConfig
	Streams           genericiooptions.IOStreams
	Resource          string
	Command           []string
	Container         string
	Filename          []string
	PodRunningTimeout time.Duration
	Quiet             bool
	Stdin             bool
	TTY               bool
}

func execCommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[execConfig] {
	cmd := cli.Command[execConfig]{
		Name: "exec",
		Description: cli.Description{
			Short: "Execute a command in a container",
			Long:  "Execute a command in a container.",
		},
		Examples: []cli.Example{
			{
				Description: "Get output from running the 'date' command from pod mypod, using the first container by default",
				CommandLine: "kubectl exec mypod -- date",
			},
			{
				Description: "Get output from running the 'date' command in ruby-container from pod mypod",
				CommandLine: "kubectl exec mypod -c ruby-container -- date",
			},
			{
				Description: "Switch to raw terminal mode; sends stdin to 'bash' in ruby-container from pod mypod\nand sends stdout/stderr from 'bash' back to the client",
				CommandLine: "kubectl exec mypod -c ruby-container -i -t -- bash -il",
			},
			{
				Description: "List contents of /usr from the first container of pod mypod and sort by modification time\nIf the command you want to execute in the pod has any flags in common (e.g. -i),\nyou must use two dashes (--) to separate your command's flags/arguments\nAlso note, do not surround your command and its flags/arguments with quotes\nunless that is how you would execute it normally (i.e., do ls -t /usr, not \"ls -t /usr\")",
				CommandLine: "kubectl exec mypod -i -t -- ls -t /usr",
			},
			{
				Description: "Get output from running 'date' command from the first pod of the deployment mydeployment, using the first container by default",
				CommandLine: "kubectl exec deploy/mydeployment -- date",
			},
			{
				Description: "Get output from running 'date' command from the first pod of the service myservice, using the first container by default",
				CommandLine: "kubectl exec svc/myservice -- date",
			},
		},
		Declare: func(cb *cli.CommandBuilder[execConfig]) {
			client := deriveClientConfig(cb, &cb.Config.Client, client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)
			cb.Dialect.EndOfOptions()
			command := cb.Values.Sequence(&cb.Config.Command, cli.Value.String())
			var resource cli.Destination[execConfig, string]
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[execConfig]) {
				resourceArgument := form.String(&form.Config.Resource, "resource")
				resource = resourceArgument.Destination()
				resourceArgument.CompleteBy(podResourceArgumentCompleter(
					func(cfg *execConfig) (clientConfig, []string) { return cfg.Client, nil },
				)).From(client)
				form.Literal("--").EndOfOptions()
				form.PassThrough(command, "command", cli.Count.OneOrMore())
			})
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[execConfig]) {
				form.Literal("--").EndOfOptions()
				form.PassThrough(command, "command", cli.Count.OneOrMore())
			})
			containerFlag := cb.Flags.String(&cb.Config.Container)
			cb.Values.ResolveWith(containerFlag, cli.Value.KeepLast())
			containerFlag.Description("Container name. If omitted, use the kubectl.kubernetes.io/default-container annotation for selecting the container to be attached or the first container in the pod will be chosen")
			containerFlag.CompleteBy(containerCompleter(
				func(cfg *execConfig) clientConfig { return cfg.Client },
				func(cfg *execConfig) string { return cfg.Resource },
			)).From(client)
			containerFlag.Long("container")
			containerFlag.Short('c')
			filenameValue := cb.Values.Sequence(&cb.Config.Filename, cli.Value.String())
			filenameFlag := cb.Flags.Sequence(filenameValue)
			filenameFlag.DelimitedBy(",")
			filenameFlag.Description("to use to exec into the resource")
			filenameFlag.Long("filename")
			filenameFlag.Short('f')
			podRunningTimeoutFlag := cb.Flags.Value(&cb.Config.PodRunningTimeout, durationLanguage())
			cb.Values.ResolveWith(podRunningTimeoutFlag, cli.Value.KeepLast())
			podRunningTimeoutFlag.Description("The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running")
			cb.Values.Default(podRunningTimeoutFlag, time.Minute)
			podRunningTimeoutFlag.Long("pod-running-timeout")
			cb.Rule.AtLeastOne(resource, filenameValue)
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
		Run: runExec,
	}
	return cmd
}

package main

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type authCanIConfig struct {
	Client        clientConfig
	Streams       genericiooptions.IOStreams
	Arguments     []string
	AllNamespaces bool
	List          bool
	NoHeaders     bool
	Quiet         bool
	Subresource   string
}

func authCanICommand(client cli.Destination[rootConfig, clientConfig]) cli.Command[authCanIConfig] {
	cmd := cli.Command[authCanIConfig]{
		Name: "can-i",
		Description: cli.Description{
			Short: "Check whether an action is allowed",
			Long:  "Check whether an action is allowed.\n\n VERB is a logical Kubernetes API verb like 'get', 'list', 'watch', 'delete', etc. TYPE is a Kubernetes resource. Shortcuts and groups will be resolved. NONRESOURCEURL is a partial URL that starts with \"/\". NAME is the name of a particular Kubernetes resource. This command pairs nicely with impersonation. See --as global flag.",
		},
		Examples: []cli.Example{
			{
				Description: "Check to see if I can create pods in any namespace",
				CommandLine: "kubectl auth can-i create pods --all-namespaces",
			},
			{
				Description: "Check to see if I can list deployments in my current namespace",
				CommandLine: "kubectl auth can-i list deployments.apps",
			},
			{
				Description: "Check to see if service account \"foo\" of namespace \"dev\" can list pods in the namespace \"prod\"\nYou must be allowed to use impersonation for the global option \"--as\"",
				CommandLine: "kubectl auth can-i list pods --as=system:serviceaccount:dev:foo -n prod",
			},
			{
				Description: "Check to see if I can do everything in my current namespace (\"*\" means all)",
				CommandLine: "kubectl auth can-i '*' '*'",
			},
			{
				Description: "Check to see if I can get the job named \"bar\" in namespace \"foo\"",
				CommandLine: "kubectl auth can-i list jobs.batch/bar -n foo",
			},
			{
				Description: "Check to see if I can read pod logs",
				CommandLine: "kubectl auth can-i get pods --subresource=log",
			},
			{
				Description: "Check to see if I can access the URL /logs/",
				CommandLine: "kubectl auth can-i get /logs/",
			},
			{
				Description: "Check to see if I can approve certificates.k8s.io",
				CommandLine: "kubectl auth can-i approve certificates.k8s.io",
			},
			{
				Description: "List all allowed actions in namespace \"foo\"",
				CommandLine: "kubectl auth can-i --list --namespace=foo",
			},
		},
		Declare: func(cb *cli.CommandBuilder[authCanIConfig]) {
			cb.Access.Value(client, &cb.Config.Client)
			cb.Access.Stdin(&cb.Config.Streams.In)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			arguments := cb.Values.Sequence(&cb.Config.Arguments, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[authCanIConfig]) {
				form.Sequence(arguments, "ARG", cli.Count.ZeroOrMore())
			})

			allNamespacesFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.AllNamespaces))
			allNamespacesFlag = allNamespacesFlag.Description("If true, check the specified action in all namespaces.")
			allNamespacesFlag.Long("all-namespaces").ImplicitValue(true)
			allNamespacesFlag.Short('A').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			listFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.List))
			listFlag = listFlag.Description("If true, prints all allowed actions.")
			listFlag.Long("list").ImplicitValue(true)

			noHeadersFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders))
			noHeadersFlag = noHeadersFlag.Description("If true, prints allowed actions without headers")
			noHeadersFlag.Long("no-headers").ImplicitValue(true)

			quietFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Quiet))
			quietFlag = quietFlag.Description("If true, suppress output and just return the exit code.")
			quietFlag.Long("quiet").ImplicitValue(true)
			quietFlag.Short('q').ValueForms(cli.FlagValueFormAssigned).ImplicitValue(true)

			subresourceFlag := cb.Flags.String(&cb.Config.Subresource)
			subresourceFlag = subresourceFlag.Description("SubResource such as pod/log or deployment/scale")
			subresourceFlag.Long("subresource")
		},
		Run: runAuthCanI,
	}
	return cmd
}

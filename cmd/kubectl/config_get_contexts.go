package main

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

type configGetContextsOutput uint8

const (
	configGetContextsOutputTable configGetContextsOutput = iota
	configGetContextsOutputName
)

var configGetContextsOutputLanguage = cli.ValueCodec[configGetContextsOutput]{
	Name: "config-get-contexts-output",
	Parse: func(text string) (configGetContextsOutput, error) {
		switch text {
		case "":
			return configGetContextsOutputTable, nil
		case "name":
			return configGetContextsOutputName, nil
		default:
			err := cli.Rejection.Message(
				cli.InputRejectionCode("unsupported-config-get-contexts-output"),
				"--output "+text+" is not available in kubectl config get-contexts; resetting to default output format",
				cli.Rejection.At(0, 0, len(text)),
			)
			return configGetContextsOutputTable, err
		}
	},
	Format: func(output configGetContextsOutput) string {
		if output == configGetContextsOutputName {
			return "name"
		}
		return ""
	},
	Snapshot: cli.Value.Assign[configGetContextsOutput](),
}

type configGetContextsConfig struct {
	Kubeconfig string
	Names      []string
	NoHeaders  bool
	Output     configGetContextsOutput
	Stdout     io.Writer
}

func configGetContextsCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configGetContextsConfig] {
	cmd := cli.Command[configGetContextsConfig]{
		Name: "get-contexts",
		Description: cli.Description{
			Short: "Describe one or many contexts",
			Long:  "Display one or many contexts from the kubeconfig file.",
		},
		Examples: []cli.Example{
			{
				Description: "List all the contexts in your kubeconfig file",
				CommandLine: "kubectl config get-contexts",
			},
			{
				Description: "Describe one context in your kubeconfig file",
				CommandLine: "kubectl config get-contexts my-context",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configGetContextsConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			names := cb.Values.Sequence(&cb.Config.Names, cli.Value.String())
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[configGetContextsConfig]) {
				form.Sequence(names, "CONTEXT_NAME", cli.Count.ZeroOrMore())
			})

			noHeaders := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders))
			cb.Values.ResolveWith(noHeaders, cli.Value.KeepLast())
			noHeaders.Description("When using the default or custom-column output format, don't print headers (default print headers).")
			noHeaders.Long("no-headers").ImplicitValue(true)

			output := cb.Values.Value(&cb.Config.Output, configGetContextsOutputLanguage)
			outputFlag := cb.Flags.ValueFor(output)
			cb.Values.ResolveWith(outputFlag, cli.Value.KeepLast())
			outputFlag.Description("Output format. One of: (name).")
			outputFlag.Long("output")
			outputFlag.Short('o')
		},
		Run: runConfigGetContexts,
	}
	return cmd
}

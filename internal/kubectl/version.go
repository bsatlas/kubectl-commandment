package kubectl

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type versionConfig struct {
	ClientConfig     clientConfig
	Streams          genericiooptions.IOStreams
	WarningsAsErrors bool
	Client           bool
	Output           string
}

func versionCommand(client cli.Destination[rootConfig, clientConfig], warningsAsErrors cli.Destination[rootConfig, bool]) cli.Command[versionConfig] {
	cmd := cli.Command[versionConfig]{
		Name: "version",
		Description: cli.Description{
			Short: "Print the client and server version information",
			Long:  "Print the client and server version information for the current context.",
		},
		Examples: []cli.Example{
			{
				Description: "Print the client and server versions for the current context",
				CommandLine: "kubectl version",
			},
		},
		Declare: func(cb *cli.CommandBuilder[versionConfig]) {
			cb.Access.Value(client, &cb.Config.ClientConfig)
			cb.Access.Value(warningsAsErrors, &cb.Config.WarningsAsErrors)
			cb.Access.Stdout(&cb.Config.Streams.Out)
			cb.Access.Stderr(&cb.Config.Streams.ErrOut)

			clientFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Client))
			clientFlag = clientFlag.Description("If true, shows client version only (no server required).")
			clientFlag.Long("client").ImplicitValue(true)

			outputFlag := cb.Flags.String(&cb.Config.Output)
			outputFlag = outputFlag.Description("One of 'yaml' or 'json'.")
			outputFlag.Long("output")
			outputFlag.Short('o')
		},
		Run: runVersion,
	}
	return cmd
}

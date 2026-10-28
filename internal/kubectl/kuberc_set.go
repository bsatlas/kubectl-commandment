package kubectl

import (
	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

type kubercSetConfig struct {
	Streams        genericiooptions.IOStreams
	AllowlistEntry []string
	Appendarg      []string
	Command        string
	Kuberc         string
	Name           string
	Option         []string
	Overwrite      bool
	Policy         string
	Prependarg     []string
	Section        string
}

var kubercSetCommand = cli.Command[kubercSetConfig]{
	Name: "set",
	Description: cli.Description{
		Short: "Set values in the kuberc configuration",
		Long:  "Set values in the kuberc configuration file.\n\n Use --section to specify whether to set defaults or aliases.\n\n For defaults: Sets default flag values for kubectl commands. The --command flag should specify only the command (e.g., \"get\", \"create\", \"set env\"), not resources.\n\n For aliases: Creates command aliases with optional default flag values and arguments. Use --prependarg and --appendarg to include resources or other arguments.",
	},
	Examples: []cli.Example{
		{
			Description: "Set default output format for 'get' command",
			CommandLine: "kubectl kuberc set --section defaults --command get --option output=wide",
		},
		{
			Description: "Set default output format for a subcommand",
			CommandLine: "kubectl kuberc set --section defaults --command \"set env\" --option output=yaml",
		},
		{
			Description: "Create an alias 'getn' for 'get' command with prepended 'nodes' resource",
			CommandLine: "kubectl kuberc set --section aliases --name getn --command get --prependarg nodes --option output=wide",
		},
		{
			Description: "Create an alias 'runx' for 'run' command with appended arguments",
			CommandLine: "kubectl kuberc set --section aliases --name runx --command run --option image=nginx --appendarg \"--\" --appendarg custom-arg1",
		},
		{
			Description: "Overwrite an existing default",
			CommandLine: "kubectl kuberc set --section defaults --command get --option output=json --overwrite",
		},
		{
			Description: "Set the credential plugin policy and allowlist",
			CommandLine: "kubectl kuberc set --section credentialplugin --policy Allowlist --allowlist-entry command=cloud-credential-helper",
		},
	},
	Declare: func(cb *cli.CommandBuilder[kubercSetConfig]) {
		cb.Access.Stdout(&cb.Config.Streams.Out)
		cb.Access.Stderr(&cb.Config.Streams.ErrOut)

		allowlistEntryValue := cb.Values.Sequence(&cb.Config.AllowlistEntry, cli.Value.String())
		allowlistEntryFlag := cb.Flags.Sequence(allowlistEntryValue)
		allowlistEntryFlag = allowlistEntryFlag.Description("Allowlist entry the form field=value (can be specified multiple times)")
		allowlistEntryFlag.Long("allowlist-entry")

		appendargValue := cb.Values.Sequence(&cb.Config.Appendarg, cli.Value.String())
		appendargFlag := cb.Flags.Sequence(appendargValue)
		appendargFlag = appendargFlag.Description("Argument to append to the command (can be specified multiple times, for aliases only)")
		appendargFlag.Long("appendarg")

		commandFlag := cb.Flags.String(&cb.Config.Command)
		commandFlag = commandFlag.Description("Command to configure (e.g., 'get', 'create', 'set env')")
		commandFlag.Long("command")

		kubercFlag := cb.Flags.String(&cb.Config.Kuberc)
		kubercFlag = kubercFlag.Description("Path to the kuberc file to use for preferences. This can be disabled by exporting KUBECTL_KUBERC=false feature gate or turning off the feature KUBERC=off.")
		kubercFlag.Long("kuberc")

		nameFlag := cb.Flags.String(&cb.Config.Name)
		nameFlag = nameFlag.Description("Alias name (required for --section=aliases)")
		nameFlag.Long("name")

		optionValue := cb.Values.Sequence(&cb.Config.Option, cli.Value.String())
		optionFlag := cb.Flags.Sequence(optionValue)
		optionFlag = optionFlag.Description("Flag option in the form flag=value (can be specified multiple times)")
		optionFlag.Long("option")

		overwriteFlag := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.Overwrite))
		overwriteFlag = overwriteFlag.Description("Allow overwriting existing entries")
		overwriteFlag.Long("overwrite").ImplicitValue(true)

		policyFlag := cb.Flags.String(&cb.Config.Policy)
		policyFlag = policyFlag.Description("Plugin policy to use for exec credential plugins, must be one of 'AllowAll', 'DenyAll' or 'Allowlist'")
		policyFlag.Long("policy")

		prependargValue := cb.Values.Sequence(&cb.Config.Prependarg, cli.Value.String())
		prependargFlag := cb.Flags.Sequence(prependargValue)
		prependargFlag = prependargFlag.Description("Argument to prepend to the command (can be specified multiple times, for aliases only)")
		prependargFlag.Long("prependarg")

		sectionFlag := cb.Flags.String(&cb.Config.Section)
		sectionFlag = sectionFlag.Description("Section to modify: 'defaults', 'aliases', or 'credentialplugin'")
		sectionFlag.Long("section")
	},
	Run: runKubercSet,
}

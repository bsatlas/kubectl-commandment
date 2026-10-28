package kubectl

import (
	"context"

	"github.com/abstractionsinc/commandment/cli"
)

var completionExamples = []cli.Example{
	{
		CommandLine: "# Installing bash completion on macOS using homebrew\n  ## If running Bash 3.2 included with macOS\n  brew install bash-completion\n  ## or, if running Bash 4.1+\n  brew install bash-completion@2\n  ## If kubectl is installed via homebrew, this should start working immediately\n  ## If you've installed via other means, you may need add the completion to your completion directory\n  kubectl completion bash > $(brew --prefix)/etc/bash_completion.d/kubectl",
	},
	{
		CommandLine: "# Installing bash completion on Linux\n  ## If bash-completion is not installed on Linux, install the 'bash-completion' package\n  ## via your distribution's package manager.\n  ## Load the kubectl completion code for bash into the current shell\n  source <(kubectl completion bash)\n  ## Write bash completion code to a file and source it from .bash_profile\n  kubectl completion bash > ~/.kube/completion.bash.inc\n  printf \"\n  # kubectl shell completion\n  source '$HOME/.kube/completion.bash.inc'\n  \" >> $HOME/.bash_profile\n  source $HOME/.bash_profile",
	},
	{
		CommandLine: "# Load the kubectl completion code for zsh[1] into the current shell\n  source <(kubectl completion zsh)\n  # Set the kubectl completion code for zsh[1] to autoload on startup\n  kubectl completion zsh > \"${fpath[1]}/_kubectl\"",
	},
	{
		CommandLine: "# Load the kubectl completion code for fish[2] into the current shell\n  kubectl completion fish | source\n  # To load completions for each session, execute once:\n  kubectl completion fish > ~/.config/fish/completions/kubectl.fish",
	},
	{
		CommandLine: "# Load the kubectl completion code for powershell into the current shell\n  kubectl completion powershell | Out-String | Invoke-Expression\n  # Set kubectl completion code for powershell to run on startup\n  ## Save completion code to a script and execute in the profile\n  kubectl completion powershell > \"$HOME\\.kube\\completion.ps1\"\n  Add-Content $PROFILE \". '$HOME\\.kube\\completion.ps1'\"\n  ## Execute completion code in the profile\n  Add-Content $PROFILE \"if (Get-Command kubectl -ErrorAction SilentlyContinue) {\n  kubectl completion powershell | Out-String | Invoke-Expression\n  }\"\n  ## Add completion code directly to the $PROFILE script\n  kubectl completion powershell >> $PROFILE",
	},
}

var (
	defaultHelp = cli.DefaultHelp(cli.DefaultHelpConfig{})
	optionsHelp = cli.DefaultHelp(cli.DefaultHelpConfig{
		Sections: []cli.HelpSection{cli.HelpSectionFlags},
	})
)

func renderHelp(ctx context.Context, page cli.HelpPage) (string, error) {
	showSensitiveFlagNames(page.FlagGroups)

	if len(page.SelectedPath) == 1 {
		switch page.SelectedPath[0] {
		case "alpha":
			return "No alpha commands are available in this version of kubectl\n", nil
		case "completion":
			page.Examples = completionExamples
		}
	}

	if len(page.SelectedPath) != 1 || page.SelectedPath[0] != "options" {
		return defaultHelp(ctx, page)
	}

	groups := page.FlagGroups[:0]
	for _, group := range page.FlagGroups {
		if group.Kind != cli.HelpGroupInherited {
			continue
		}
		flags := group.Flags[:0]
		for _, flag := range group.Flags {
			if flag.ActionEntry != 0 {
				continue
			}
			flags = append(flags, flag)
		}
		if len(flags) == 0 {
			continue
		}
		group.Flags = flags
		group.Title = "The following options can be passed to any command"
		groups = append(groups, group)
	}
	page.FlagGroups = groups
	return optionsHelp(ctx, page)
}

func showSensitiveFlagNames(groups []cli.HelpFlagGroup) {
	for groupIndex := range groups {
		for flagIndex := range groups[groupIndex].Flags {
			groups[groupIndex].Flags[flagIndex].Sensitive = false
		}
	}
}

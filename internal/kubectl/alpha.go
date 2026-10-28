package kubectl

import "github.com/abstractionsinc/commandment/cli"

var alphaCommand = cli.Command[struct{}]{
	Name: "alpha",
	Description: cli.Description{
		Short: "Commands for features in alpha",
		Long:  "These commands correspond to alpha features that are not enabled in Kubernetes clusters by default.",
	},
	Hidden: true,
}

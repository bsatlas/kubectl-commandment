package kubectl

import "github.com/abstractionsinc/commandment/cli"

var kubectlRelease = cli.Release.Ordered("v1.38", "pre-v1.38", "v1.38")

func preexistingDeprecation(reason string) cli.ReleaseLifecycle {
	lifecycle := cli.ReleaseLifecycle{
		DeprecatedIn: "pre-v1.38",
		Reason:       reason,
	}
	return lifecycle
}

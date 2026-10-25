package kubectlcommandment_test

import (
	"io"

	"github.com/abstractionsinc/commandment/cli"
)

func runRequest(args []string, stdout io.Writer) cli.RunRequest {
	return cli.RunRequest{
		Args:   args,
		Stdout: stdout,
		Stderr: io.Discard,
	}
}

package kubectl

import (
	"errors"

	"github.com/abstractionsinc/commandment/cli"
)

var errSilentExit = errors.New("kubectl requested a nonzero exit status")

func silentExit(code int) error {
	return cli.WithExitCode(errSilentExit, code)
}

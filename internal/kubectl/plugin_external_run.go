package kubectl

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/abstractionsinc/commandment/cli"
)

type externalPluginConfig struct {
	Arguments []string
}

func runExternalPlugin(lookupName string) cli.Handler[externalPluginConfig] {
	return func(ctx context.Context, cfg *externalPluginConfig) error {
		path, err := exec.LookPath("kubectl-" + lookupName)
		if err != nil {
			return err
		}
		environment, err := pluginEnvironment()
		if err != nil {
			return err
		}
		if runtime.GOOS == "windows" {
			command := exec.CommandContext(ctx, path, cfg.Arguments...)
			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			command.Env = environment
			return command.Run()
		}
		return syscall.Exec(path, append([]string{path}, cfg.Arguments...), environment)
	}
}

func pluginEnvironment() ([]string, error) {
	current := os.Environ()
	environment := make([]string, 0, len(current)+1)
	for _, value := range current {
		if strings.HasPrefix(value, "KUBECTL_PATH=") {
			continue
		}
		environment = append(environment, value)
	}
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return append(environment, "KUBECTL_PATH="+path), nil
}

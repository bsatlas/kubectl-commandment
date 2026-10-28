package kubectl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"k8s.io/kubectl/pkg/cmd/plugin"
)

type pluginPathVerifier struct {
	seen map[string]string
}

func (verifier *pluginPathVerifier) Verify(path string) []error {
	var result []error

	executable, err := pluginExecutable(path)
	if err != nil {
		result = append(result, fmt.Errorf("error: unable to identify %s as an executable file: %v", path, err))
	} else if !executable {
		result = append(result, fmt.Errorf("warning: %s identified as a kubectl plugin, but it is not executable", path))
	}

	name := filepath.Base(path)
	if existing, ok := verifier.seen[name]; ok {
		result = append(result, fmt.Errorf("warning: %s is overshadowed by a similarly named plugin: %s", path, existing))
	} else {
		verifier.seen[name] = path
	}

	return result
}

func runPluginList(_ context.Context, cfg *pluginListConfig) error {
	options := &plugin.PluginListOptions{
		Verifier: &pluginPathVerifier{
			seen: make(map[string]string),
		},
		NameOnly:    cfg.NameOnly,
		PluginPaths: filepath.SplitList(os.Getenv("PATH")),
		IOStreams:   cfg.Streams,
	}
	return options.Run()
}

func pluginExecutable(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if runtime.GOOS == "windows" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".bat", ".cmd", ".com", ".exe", ".ps1":
			return true, nil
		}
		return false, nil
	}
	return !info.IsDir() && info.Mode()&0o111 != 0, nil
}

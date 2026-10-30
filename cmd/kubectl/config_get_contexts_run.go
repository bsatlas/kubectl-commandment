package main

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/tools/clientcmd/api"
)

func runConfigGetContexts(_ context.Context, cfg *configGetContextsConfig) error {
	config, err := configAccess(cfg.Kubeconfig).GetStartingConfig()
	if err != nil {
		return err
	}

	out := printers.GetNewTabWriter(cfg.Stdout)
	defer out.Flush()

	var allErrs []error
	toPrint := make([]string, 0, len(config.Contexts))
	if len(cfg.Names) == 0 {
		for name := range config.Contexts {
			toPrint = append(toPrint, name)
		}
	} else {
		for _, name := range cfg.Names {
			if _, ok := config.Contexts[name]; ok {
				toPrint = append(toPrint, name)
			} else {
				allErrs = append(allErrs, fmt.Errorf("context %v not found", name))
			}
		}
	}

	nameOnly := cfg.Output == configGetContextsOutputName
	showHeaders := !cfg.NoHeaders && !nameOnly
	if showHeaders {
		if err := printContextHeaders(out, nameOnly); err != nil {
			allErrs = append(allErrs, err)
		}
	}

	slices.Sort(toPrint)
	for _, name := range toPrint {
		if err := printContext(name, config.Contexts[name], out, nameOnly, config.CurrentContext == name); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	return errors.NewAggregate(allErrs)
}

func printContextHeaders(out io.Writer, nameOnly bool) error {
	columns := []string{"CURRENT", "NAME", "CLUSTER", "AUTHINFO", "NAMESPACE"}
	if nameOnly {
		columns = columns[:1]
	}
	_, err := fmt.Fprintf(out, "%s\n", strings.Join(columns, "\t"))
	return err
}

func printContext(name string, currentContext *api.Context, out io.Writer, nameOnly, current bool) error {
	if nameOnly {
		_, err := fmt.Fprintln(out, name)
		return err
	}
	prefix := " "
	if current {
		prefix = "*"
	}
	_, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", prefix, name, currentContext.Cluster, currentContext.AuthInfo, currentContext.Namespace)
	return err
}

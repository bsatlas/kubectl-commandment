package main

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/abstractionsinc/commandment/cli"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/tools/clientcmd/api"
)

type configGetContextsConfig struct {
	Kubeconfig string
	Names      []string
	NoHeaders  bool
	Output     string
	Stdout     io.Writer
}

func configGetContextsCommand(kubeconfig cli.Destination[rootConfig, string]) cli.Command[configGetContextsConfig] {
	cmd := cli.Command[configGetContextsConfig]{
		Name: "get-contexts",
		Description: cli.Description{
			Short: "Describe one or many contexts",
			Long:  "Display one or many contexts from the kubeconfig file.",
		},
		Examples: []cli.Example{
			{
				Description: "List all the contexts in your kubeconfig file",
				CommandLine: "kubectl config get-contexts",
			},
			{
				Description: "Describe one context in your kubeconfig file",
				CommandLine: "kubectl config get-contexts my-context",
			},
		},
		Declare: func(cb *cli.CommandBuilder[configGetContextsConfig]) {
			cb.Access.Value(kubeconfig, &cb.Config.Kubeconfig)
			cb.Access.Stdout(&cb.Config.Stdout)

			contexts := cb.Values.Domain(func(_ context.Context, _ *configGetContextsConfig, dependencies cli.Dependencies, domain *cli.ValueDomainBuilder[string]) error {
				path, ok := dependencies.Get(kubeconfig)
				if !ok {
					return fmt.Errorf("kubeconfig dependency is unavailable")
				}
				names, err := contextNames(path)
				if err != nil {
					return err
				}
				for _, name := range names {
					domain.Value(name, name)
				}
				return nil
			}, kubeconfig)
			names := cb.Values.Sequence(&cb.Config.Names, contexts)
			cb.Arguments.Form(func(form *cli.ArgumentFormBuilder[configGetContextsConfig]) {
				form.Sequence(names, "CONTEXT_NAME", cli.Count.ZeroOrMore())
			})

			noHeaders := cb.Flags.ValueFor(cb.Values.Bool(&cb.Config.NoHeaders)).Description("When using the default or custom-column output format, don't print headers (default print headers).")
			noHeaders.Long("no-headers").ImplicitValue(true)

			output := cb.Flags.String(&cb.Config.Output).Description("Output format. One of: (name).")
			output.Long("output")
			output.Short('o')
			cb.Rule.In(output, "", "name")
		},
		Run: runConfigGetContexts,
	}
	return cmd
}

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

	nameOnly := cfg.Output == "name"
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

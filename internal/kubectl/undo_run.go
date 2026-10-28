package kubectl

import (
	"context"

	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/rollout"
	"k8s.io/kubectl/pkg/cmd/util"
)

func runRolloutUndo(_ context.Context, cfg *rolloutUndoConfig) error {
	factory := cfg.Client.factory()
	options := rollout.NewRolloutUndoOptions(cfg.Streams)
	options.Resources = cfg.Resources.Args()
	options.ToRevision = cfg.ToRevision
	options.DryRunStrategy = cfg.DryRun
	options.LabelSelector = cfg.Selector
	options.FilenameOptions = resource.FilenameOptions{
		Filenames: cfg.Filename.Filenames(),
		Kustomize: cfg.Filename.Kustomize,
		Recursive: cfg.Filename.Recursive,
	}
	options.CmdBaseName = "kubectl"
	configurePrintFlagsWithOutputState(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	if options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace(); err != nil {
		return err
	}
	options.ToPrinter = func(operation string) (printers.ResourcePrinter, error) {
		options.PrintFlags.NamePrintFlags.Operation = operation
		util.PrintFlagsWithDryRunStrategy(options.PrintFlags, options.DryRunStrategy)
		return options.PrintFlags.ToPrinter()
	}
	options.RESTClientGetter = factory
	options.Builder = factory.NewBuilder
	return options.RunUndo()
}

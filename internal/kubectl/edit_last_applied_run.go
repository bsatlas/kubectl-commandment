package kubectl

import "context"

func runApplyEditLastApplied(_ context.Context, cfg *applyEditLastAppliedConfig) error {
	runConfig := editRunConfig{
		Client:                   cfg.Client,
		Streams:                  cfg.Streams,
		Arguments:                cfg.Arguments,
		AllowMissingTemplateKeys: cfg.AllowMissingTemplateKeys,
		FieldManager:             cfg.FieldManager,
		Filename:                 cfg.Filename,
		Kustomize:                cfg.Kustomize,
		Output:                   cfg.Output,
		Record:                   cfg.Record,
		Recursive:                cfg.Recursive,
		ShowManagedFields:        cfg.ShowManagedFields,
		Template:                 cfg.Template,
		Validate:                 cfg.Validate,
		WindowsLineEndings:       cfg.WindowsLineEndings,
	}
	return runEditOperation(runConfig, editModeApply)
}

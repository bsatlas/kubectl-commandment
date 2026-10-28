package main

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/rawhttp"
	"k8s.io/kubectl/pkg/scheme"
)

func runCreate(_ context.Context, cfg *createConfig) error {
	if len(cfg.Arguments) != 0 {
		return fmt.Errorf("unexpected args: %v", cfg.Arguments)
	}
	filename := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	if err := filename.RequireFilenameOrKustomize(); err != nil {
		return err
	}
	if cfg.Raw != "" {
		if cfg.Edit {
			return fmt.Errorf("--raw and --edit are mutually exclusive")
		}
		if len(filename.Filenames) != 1 {
			return fmt.Errorf("--raw can only use a single local file or stdin")
		}
		if strings.HasPrefix(filename.Filenames[0], "http://") || strings.HasPrefix(filename.Filenames[0], "https://") {
			return fmt.Errorf("--raw cannot read from a url")
		}
		if filename.Recursive {
			return fmt.Errorf("--raw and --recursive are mutually exclusive")
		}
		if cfg.Selector != "" {
			return fmt.Errorf("--raw and --selector (-l) are mutually exclusive")
		}
		if cfg.Output != "" {
			return fmt.Errorf("--raw and --output are mutually exclusive")
		}
		if _, err := url.ParseRequestURI(cfg.Raw); err != nil {
			return fmt.Errorf("--raw must be a valid URL path: %v", err)
		}
	}

	strategy, err := dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	factory := cfg.Client.factory()
	if cfg.Raw != "" {
		client, err := factory.RESTClient()
		if err != nil {
			return err
		}
		return rawhttp.RawPost(client, cfg.Streams, cfg.Raw, filename.Filenames[0])
	}
	if cfg.Edit {
		return runCreateEdit(cfg, factory, filename, strategy)
	}

	validation, err := factory.Validator(cfg.Validate)
	if err != nil {
		return err
	}
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	util.PrintFlagsWithDryRunStrategy(printFlags, strategy)
	printer, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl create"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	result := factory.NewBuilder().
		Unstructured().
		Schema(validation).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		LabelSelectorParam(cfg.Selector).
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	count := 0
	err = result.Visit(func(info *resource.Info, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if err := saveConfigAnnotation(cfg.SaveConfig, info.Object); err != nil {
			return util.AddSourceToErr("creating", info.Source, err)
		}
		if err := recorder.Record(info.Object); err != nil {
			klog.V(4).Infof("error recording current command: %v", err)
		}
		if strategy != util.DryRunClient {
			object, err := resource.NewHelper(info.Client, info.Mapping).
				DryRun(strategy == util.DryRunServer).
				WithFieldManager(cfg.FieldManager).
				WithFieldValidation(cfg.Validate).
				Create(info.Namespace, true, info.Object)
			if err != nil {
				return util.AddSourceToErr("creating", info.Source, err)
			}
			info.Refresh(object, true)
		}
		count++
		return printer.PrintObj(info.Object, cfg.Streams.Out)
	})
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("no objects passed to create")
	}
	return nil
}

func runCreateEdit(cfg *createConfig, factory util.Factory, filename resource.FilenameOptions, strategy util.DryRunStrategy) error {
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	result := factory.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).
		DefaultNamespace().
		FilenameParam(enforceNamespace, &filename).
		ContinueOnError().
		Flatten().
		Do()
	if err := result.Err(); err != nil {
		return err
	}

	validation, err := validationDirective(cfg.Validate)
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)
	configurePrintFlags(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	util.PrintFlagsWithDryRunStrategy(printFlags, strategy)
	toPrinter := func(operation string) (printers.ResourcePrinter, error) {
		printFlags.NamePrintFlags.Operation = operation
		return printFlags.ToPrinter()
	}

	editorFlags := (&genericclioptions.PrintFlags{
		JSONYamlPrintFlags: genericclioptions.NewJSONYamlPrintFlags(),
	}).WithDefaultOutput("yaml")
	editorPrinter := &editPrinterOptions{
		printFlags: editorFlags,
		ext:        ".yaml",
		addHeader:  true,
	}
	if err := editorPrinter.Complete(printFlags); err != nil {
		return err
	}

	recordFlags := genericclioptions.NewRecordFlags()
	*recordFlags.Record = cfg.Record
	if cfg.Record {
		if err := recordFlags.CompleteWithChangeCause("kubectl create"); err != nil {
			return err
		}
	}
	recorder, err := recordFlags.ToRecorder()
	if err != nil {
		return err
	}

	operation := editOperation{
		ToPrinter:           toPrinter,
		WindowsLineEndings:  cfg.WindowsLineEndings,
		ValidationDirective: validation,
		OriginalResult:      result,
		Mode:                editModeBeforeCreate,
		CmdNamespace:        namespace,
		ApplyAnnotation:     cfg.SaveConfig,
		IOStreams:           cfg.Streams,
		Recorder:            recorder,
		f:                   factory,
		editPrinterOptions:  editorPrinter,
		updatedResultGetter: func(data []byte) *resource.Result {
			return factory.NewBuilder().
				Unstructured().
				Stream(bytes.NewReader(data), "edited-file").
				ContinueOnError().
				Flatten().
				Do()
		},
		FieldManager: "kubectl-create",
	}
	return operation.Run()
}

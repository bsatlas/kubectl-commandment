package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/create"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/rawhttp"
	"k8s.io/kubectl/pkg/scheme"
)

func runCreate(ctx context.Context, cfg *createConfig) error {
	filename := resource.FilenameOptions{
		Filenames: slices.Concat(cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	strategy := cfg.DryRun
	factory := cfg.Client.factoryForInvocation(ctx)
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

	validationDirective, err := validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	validation, err := factory.Validator(validationDirective)
	if err != nil {
		return err
	}
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	printFlags := genericclioptions.NewPrintFlags("created").WithTypeSetter(scheme.Scheme)
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
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

	builder := factory.NewBuilder().
		Unstructured().
		Schema(validation).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace()
	addFilenameSources(builder, enforceNamespace, filename, cfg.Streams.In)
	result := builder.
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
				WithFieldValidation(validationDirective).
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

	builder := factory.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).
		DefaultNamespace().
		ContinueOnError()
	addFilenameSources(builder, enforceNamespace, filename, cfg.Streams.In)
	result := builder.
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
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
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

func standardCreateRoleVerb(verb string) bool {
	switch verb {
	case "*", "get", "delete", "list", "create", "update", "patch", "watch", "proxy", "deletecollection", "use", "bind", "escalate", "impersonate":
		return true
	default:
		return false
	}
}

func warnCreateRoleVerbs(out io.Writer, verbs []string) {
	for _, verb := range verbs {
		if !standardCreateRoleVerb(verb) {
			fmt.Fprintf(out, "Warning: '%s' is not a standard resource verb\n", verb)
		}
	}
}

func createRoleSpecialVerbAllows(verb string, resource schema.GroupVersionResource) (special, allowed bool) {
	switch verb {
	case "use":
		allowed := resource.Resource == "podsecuritypolicies" && (resource.Group == "policy" || resource.Group == "extensions")
		return true, allowed
	case "bind", "escalate":
		allowed := resource.Group == "rbac.authorization.k8s.io" && (resource.Resource == "roles" || resource.Resource == "clusterroles")
		return true, allowed
	case "impersonate":
		if resource.Group == "" {
			return true, resource.Resource == "users" || resource.Resource == "serviceaccounts" || resource.Resource == "groups"
		}
		return true, resource.Group == "authentication.k8s.io" && resource.Resource == "userextras"
	default:
		return false, false
	}
}

func validateCreateRoleResources(mapper meta.RESTMapper, verbs []string, resources []create.ResourceOptions) error {
	for _, requested := range resources {
		if requested.Resource == "*" {
			return nil
		}

		resource := schema.GroupVersionResource{
			Resource: requested.Resource,
			Group:    requested.Group,
		}
		resolved, err := mapper.ResourceFor(resource)
		if err == nil {
			resource = resolved
		}

		for _, verb := range verbs {
			special, allowed := createRoleSpecialVerbAllows(verb, resource)
			if !special {
				continue
			}
			if !allowed {
				return fmt.Errorf("can not perform '%s' on '%s' in group '%s'", verb, resource.Resource, resource.Group)
			}
			err = nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

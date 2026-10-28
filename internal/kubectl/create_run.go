package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	policyv1 "k8s.io/client-go/kubernetes/typed/policy/v1"
	schedulingv1 "k8s.io/client-go/kubernetes/typed/scheduling/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/create"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/rawhttp"
	"k8s.io/kubectl/pkg/scheme"
)

func runCreate(_ context.Context, cfg *createConfig) error {
	filename := resource.FilenameOptions{
		Filenames: append([]string(nil), cfg.Filename...),
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
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

func runCreateConfigMap(_ context.Context, cfg *createConfigMapConfig) error {
	factory := cfg.Client.factory()
	options := create.NewConfigMapOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FileSources = append([]string(nil), cfg.FromFile...)
	options.LiteralSources = append([]string(nil), cfg.FromLiteral...)
	options.EnvFileSources = append([]string(nil), cfg.FromEnvFile...)
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateSecretGeneric(_ context.Context, cfg *createSecretGenericConfig) error {
	factory := cfg.Client.factory()
	options := create.NewSecretOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Type = cfg.Type
	options.FileSources = append([]string(nil), cfg.FromFile...)
	options.LiteralSources = append([]string(nil), cfg.FromLiteral...)
	options.EnvFileSources = append([]string(nil), cfg.FromEnvFile...)
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateSecretDockerRegistry(_ context.Context, cfg *createSecretDockerRegistryConfig) error {
	factory := cfg.Client.factory()
	options := create.NewSecretDockerRegistryOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FileSources = append([]string(nil), cfg.FromFile...)
	options.Username = cfg.DockerUsername
	options.Email = cfg.DockerEmail
	options.Password = cfg.DockerPassword
	options.Server = cfg.DockerServer
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateSecretTLS(_ context.Context, cfg *createSecretTLSConfig) error {
	factory := cfg.Client.factory()
	options := create.NewSecretTLSOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Key = cfg.Key
	options.Cert = cfg.Cert
	options.AppendHash = cfg.AppendHash
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceClusterIP(_ context.Context, cfg *createServiceClusterIPConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, "ClusterIP")
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
	options.ClusterIP = cfg.ClusterIP
	options.FieldManager = cfg.FieldManager

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceExternalName(_ context.Context, cfg *createServiceExternalNameConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, "ExternalName")
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
	options.ExternalName = cfg.ExternalName
	options.FieldManager = cfg.FieldManager

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceLoadBalancer(_ context.Context, cfg *createServiceLoadBalancerConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, "LoadBalancer")
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
	options.FieldManager = cfg.FieldManager

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceNodePort(_ context.Context, cfg *createServiceNodePortConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceOptions(cfg.Streams, "NodePort")
	options.Name = cfg.Name
	options.TCP = append([]string(nil), cfg.TCP...)
	options.NodePort = cfg.NodePort
	options.FieldManager = cfg.FieldManager

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.CoreV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateNamespace(_ context.Context, cfg *createNamespaceConfig) error {
	factory := cfg.Client.factory()
	options := create.NewNamespaceOptions(cfg.Streams)
	options.Name = cfg.Name
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateQuota(_ context.Context, cfg *createQuotaConfig) error {
	factory := cfg.Client.factory()
	options := create.NewQuotaOpts(cfg.Streams)
	options.Name = cfg.Name
	options.Hard = cfg.Hard
	options.Scopes = cfg.Scopes
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateServiceAccount(_ context.Context, cfg *createServiceAccountConfig) error {
	factory := cfg.Client.factory()
	options := create.NewServiceAccountOpts(cfg.Streams)
	options.Name = cfg.Name
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = corev1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreatePodDisruptionBudget(_ context.Context, cfg *createPodDisruptionBudgetConfig) error {
	factory := cfg.Client.factory()
	options := create.NewPodDisruptionBudgetOpts(cfg.Streams)
	options.Name = cfg.Name
	options.MinAvailable = cfg.MinAvailable
	options.MaxUnavailable = cfg.MaxUnavailable
	options.Selector = cfg.Selector
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = policyv1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
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

func createRoleSpecialVerbAllows(verb string, resource schema.GroupVersionResource) (special bool, allowed bool) {
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

func runCreateRole(_ context.Context, cfg *createRoleConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateRoleOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Verbs = append([]string(nil), cfg.Verbs...)
	options.Resources = append([]create.ResourceOptions(nil), cfg.Resources...)
	options.ResourceNames = append([]string(nil), cfg.ResourceNames...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig
	options.OutputFormat = cfg.Output

	var err error
	options.Mapper, err = factory.ToRESTMapper()
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.RbacV1()
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	warnCreateRoleVerbs(options.ErrOut, options.Verbs)
	if err := validateCreateRoleResources(options.Mapper, options.Verbs, options.Resources); err != nil {
		return err
	}
	return options.RunCreateRole()
}

func runCreateClusterRole(_ context.Context, cfg *createClusterRoleConfig) error {
	factory := cfg.Client.factory()
	role := create.NewCreateRoleOptions(cfg.Streams)
	role.Name = cfg.Name
	role.Verbs = append([]string(nil), cfg.Verbs...)
	role.Resources = append([]create.ResourceOptions(nil), cfg.Resources...)
	role.ResourceNames = append([]string(nil), cfg.ResourceNames...)
	role.CreateAnnotation = cfg.SaveConfig
	role.OutputFormat = cfg.Output

	options := &create.CreateClusterRoleOptions{
		CreateRoleOptions: role,
		NonResourceURLs:   append([]string(nil), cfg.NonResourceURLs...),
		AggregationRule:   cfg.AggregationRule,
		FieldManager:      cfg.FieldManager,
	}

	var err error
	options.Mapper, err = factory.ToRESTMapper()
	if err != nil {
		return err
	}
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.RbacV1()
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	if len(options.Resources) > 0 {
		warnCreateRoleVerbs(options.ErrOut, options.Verbs)
		if err := validateCreateRoleResources(options.Mapper, options.Verbs, options.Resources); err != nil {
			return err
		}
	}
	return options.RunCreateRole()
}

func runCreateRoleBinding(_ context.Context, cfg *createRoleBindingConfig) error {
	factory := cfg.Client.factory()
	options := create.NewRoleBindingOptions(cfg.Streams)
	options.Name = cfg.Name
	options.ClusterRole = cfg.ClusterRole
	options.Role = cfg.Role
	options.Users = append([]string(nil), cfg.Users...)
	options.Groups = append([]string(nil), cfg.Groups...)
	options.ServiceAccounts = append([]string(nil), cfg.ServiceAccounts...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	var err error
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.RbacV1()
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateClusterRoleBinding(_ context.Context, cfg *createClusterRoleBindingConfig) error {
	factory := cfg.Client.factory()
	options := create.NewClusterRoleBindingOptions(cfg.Streams)
	options.Name = cfg.Name
	options.ClusterRole = cfg.ClusterRole
	options.Users = append([]string(nil), cfg.Users...)
	options.Groups = append([]string(nil), cfg.Groups...)
	options.ServiceAccounts = append([]string(nil), cfg.ServiceAccounts...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.RbacV1()
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreatePriorityClass(_ context.Context, cfg *createPriorityClassConfig) error {
	factory := cfg.Client.factory()
	options := create.NewPriorityClassOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Value = cfg.Value
	options.GlobalDefault = cfg.GlobalDefault
	options.Description = cfg.Description
	options.PreemptionPolicy = cfg.PreemptionPolicy
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	options.Client, err = schedulingv1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateToken(_ context.Context, cfg *createTokenConfig) error {
	factory := cfg.Client.factory()
	options := create.NewTokenOpts(cfg.Streams)
	options.Name = cfg.Name
	options.BoundObjectKind = cfg.BoundObjectKind
	options.BoundObjectName = cfg.BoundObjectName
	options.BoundObjectUID = cfg.BoundObjectUID
	options.Audiences = append([]string(nil), cfg.Audiences...)
	options.Duration = cfg.Duration

	var err error
	options.Namespace, _, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.CoreClient = clientset.CoreV1()
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, util.DryRunNone, cfg.Streams.Out)
	if err != nil {
		return err
	}
	if options.CoreClient == nil {
		return fmt.Errorf("no client provided")
	}
	if options.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	return options.Run()
}

func runCreateDeployment(_ context.Context, cfg *createDeploymentConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateDeploymentOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Images = append([]string(nil), cfg.Images...)
	options.Port = cfg.Port
	options.Replicas = cfg.Replicas
	options.Command = append([]string(nil), cfg.Command...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.AppsV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateCronJob(_ context.Context, cfg *createCronJobConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateCronJobOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Image = cfg.Image
	options.Schedule = cfg.Schedule
	options.Command = append([]string(nil), cfg.Command...)
	options.Restart = cfg.Restart
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig
	options.Builder = factory.NewBuilder()

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.BatchV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateJob(_ context.Context, cfg *createJobConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateJobOptions(cfg.Streams)
	options.Name = cfg.Name
	options.Image = cfg.Image
	options.From = cfg.From
	options.Command = append([]string(nil), cfg.Command...)
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig
	options.Builder = factory.NewBuilder()

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.BatchV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

func runCreateIngress(_ context.Context, cfg *createIngressConfig) error {
	factory := cfg.Client.factory()
	options := create.NewCreateIngressOptions(cfg.Streams)
	options.Name = cfg.Name
	options.IngressClass = cfg.Class
	options.Rules = append([]string(nil), cfg.Rules...)
	options.Annotations = append([]string(nil), cfg.Annotations...)
	options.DefaultBackend = cfg.DefaultBackend
	options.FieldManager = cfg.FieldManager
	options.CreateAnnotation = cfg.SaveConfig

	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.Client = clientset.NetworkingV1()
	options.Namespace, options.EnforceNamespace, err = factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.DryRunStrategy, err = dryRunStrategy(cfg.DryRun)
	if err != nil {
		return err
	}
	options.ValidationDirective, err = validationDirective(cfg.Validate)
	if err != nil {
		return err
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	options.PrintObj, err = printObject(options.PrintFlags, options.DryRunStrategy, cfg.Streams.Out)
	if err != nil {
		return err
	}
	return options.Run()
}

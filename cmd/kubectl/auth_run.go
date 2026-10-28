package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/kubectl/pkg/cmd/auth"
	"k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

func runAuthCanI(_ context.Context, cfg *authCanIConfig) error {
	factory := cfg.Client.factory()
	streams := genericiooptions.IOStreams{
		Out:    cfg.Stdout,
		ErrOut: cfg.Stderr,
	}
	warningOptions := printers.WarningPrinterOptions{
		Color: printers.AllowsColorOutput(cfg.Stderr),
	}
	warningPrinter := printers.NewWarningPrinter(cfg.Stderr, warningOptions)
	options := &auth.CanIOptions{
		AllNamespaces:  cfg.AllNamespaces,
		Quiet:          cfg.Quiet,
		NoHeaders:      cfg.NoHeaders,
		Subresource:    cfg.Subresource,
		List:           cfg.List,
		IOStreams:      streams,
		WarningPrinter: warningPrinter,
	}
	if !cfg.List {
		options.Verb = cfg.Verb
		if cfg.Target.NonResourceURL != "" {
			options.NonResourceURL = cfg.Target.NonResourceURL
		} else {
			mapper, err := factory.ToRESTMapper()
			if err != nil {
				return err
			}
			options.Resource = resolveAuthCanIResource(mapper, cfg.Target.Resource, options.WarningPrinter)
			options.ResourceName = cfg.Target.ResourceName
		}
		if cfg.Quiet {
			options.Out = io.Discard
		}
	}

	client, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.AuthClient = client.AuthorizationV1()
	options.DiscoveryClient = client.Discovery()
	if !cfg.AllNamespaces {
		options.Namespace, _, err = factory.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}
	}
	warnAuthCanI(options)
	if options.List {
		return options.RunAccessList()
	}
	allowed, err := options.RunAccessCheck()
	if err != nil {
		return err
	}
	if !allowed {
		return silentExit(1)
	}
	return nil
}

func warnAuthCanI(options *auth.CanIOptions) {
	if options.NonResourceURL != "" {
		if !knownAuthCanINonResourceVerb(options.Verb) {
			options.WarningPrinter.Print(fmt.Sprintf("verb '%s' is not a known verb\n", options.Verb))
		}
		return
	}
	if options.Resource.Empty() || options.AllNamespaces || options.DiscoveryClient == nil {
		return
	}

	namespaced, err := authCanIResourceNamespaced(options.Resource, options.DiscoveryClient)
	if err == nil && !namespaced {
		if options.Resource.Group == "" {
			options.WarningPrinter.Print(fmt.Sprintf("resource '%s' is not namespace scoped\n", options.Resource.Resource))
		} else {
			options.WarningPrinter.Print(fmt.Sprintf("resource '%s' is not namespace scoped in group '%s'\n", options.Resource.Resource, options.Resource.Group))
		}
	}
	if !knownAuthCanIResourceVerb(options.Verb) {
		options.WarningPrinter.Print(fmt.Sprintf("verb '%s' is not a known verb\n", options.Verb))
	}
}

func authCanIResourceNamespaced(resource schema.GroupVersionResource, client discovery.DiscoveryInterface) (bool, error) {
	if resource.Resource == "*" {
		return true, nil
	}
	groupVersion := schema.GroupVersion{
		Group:   resource.Group,
		Version: resource.Version,
	}
	resources, err := client.ServerResourcesForGroupVersion(groupVersion.String())
	if err != nil {
		return true, err
	}
	for _, candidate := range resources.APIResources {
		if candidate.Name == resource.Resource {
			return candidate.Namespaced, nil
		}
	}
	return false, fmt.Errorf("the server doesn't have a resource type '%s' in group '%s'", resource.Resource, resource.Group)
}

func knownAuthCanIResourceVerb(verb string) bool {
	switch verb {
	case "get", "list", "watch", "create", "update", "patch", "delete", "deletecollection", "use", "bind", "impersonate", "*", "approve", "sign", "escalate", "attest":
		return true
	default:
		return false
	}
}

func knownAuthCanINonResourceVerb(verb string) bool {
	switch verb {
	case "get", "put", "post", "head", "options", "delete", "patch", "*":
		return true
	default:
		return false
	}
}

func resolveAuthCanIResource(mapper meta.RESTMapper, resourceArg string, warnings *printers.WarningPrinter) schema.GroupVersionResource {
	if resourceArg == "*" {
		resource := schema.GroupVersionResource{
			Resource: resourceArg,
		}
		return resource
	}

	fullySpecified, groupResource := schema.ParseResourceArg(strings.ToLower(resourceArg))
	var resource schema.GroupVersionResource
	if fullySpecified != nil {
		resource, _ = mapper.ResourceFor(*fullySpecified)
	}
	if !resource.Empty() {
		return resource
	}

	resource, err := mapper.ResourceFor(groupResource.WithVersion(""))
	if err == nil {
		return resource
	}
	if groupResource.String() != "users" && groupResource.String() != "groups" {
		if groupResource.Group == "" {
			warnings.Print(fmt.Sprintf("the server doesn't have a resource type '%s'\n", groupResource.Resource))
		} else {
			warnings.Print(fmt.Sprintf("the server doesn't have a resource type '%s' in group '%s'\n", groupResource.Resource, groupResource.Group))
		}
	}
	fallback := schema.GroupVersionResource{
		Resource: resourceArg,
	}
	return fallback
}

func runAuthReconcile(_ context.Context, cfg *authReconcileConfig) error {
	factory := cfg.Client.factory()
	streams := genericiooptions.IOStreams{
		Out:    cfg.Stdout,
		ErrOut: cfg.Stderr,
	}
	options := auth.NewReconcileOptions(streams)
	options.FilenameOptions = &resource.FilenameOptions{
		Filenames: cfg.Filename,
		Kustomize: cfg.Kustomize,
		Recursive: cfg.Recursive,
	}
	options.DryRun = cfg.DryRun == util.DryRunClient
	options.RemoveExtraPermissions = cfg.RemoveExtraPermissions
	options.RemoveExtraSubjects = cfg.RemoveExtraSubjects

	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	result := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		ContinueOnError().
		NamespaceParam(namespace).DefaultNamespace().
		FilenameParam(enforceNamespace, options.FilenameOptions).
		Flatten().
		Local().
		Do()
	if err := result.Err(); err != nil {
		return err
	}
	options.Visitor = result
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	options.RBACClient = clientset.RbacV1()
	options.NamespaceClient = clientset.CoreV1()
	if options.DryRun {
		if err := options.PrintFlags.Complete("%s (dry run)"); err != nil {
			return err
		}
	}
	configurePrintFlags(options.PrintFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields)
	printer, err := options.PrintFlags.ToPrinter()
	if err != nil {
		return err
	}
	options.PrintObject = printer.PrintObj
	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunReconcile()
}

func runAuthWhoAmI(ctx context.Context, cfg *authWhoAmIConfig) error {
	streams := genericiooptions.IOStreams{
		Out: cfg.Stdout,
	}
	flags := auth.NewWhoAmIFlags(restClientGetter(cfg.Client), streams)
	*flags.PrintFlags.OutputFormat = cfg.Output
	flags.PrintFlags.OutputFlagSpecified = func() bool { return flagProvided(cfg.OutputState) }
	flags.PrintFlags.JSONYamlPrintFlags.ShowManagedFields = cfg.ShowManagedFields
	*flags.PrintFlags.TemplatePrinterFlags.TemplateArgument = cfg.Template
	*flags.PrintFlags.TemplatePrinterFlags.AllowMissingKeys = cfg.AllowMissingTemplateKeys
	options, err := flags.ToOptions(ctx, nil)
	if err != nil {
		return err
	}
	return options.Run()
}

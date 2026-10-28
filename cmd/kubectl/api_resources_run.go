package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubectl/pkg/cmd/apiresources"
)

func runAPIResources(_ context.Context, cfg *apiResourcesConfig) error {
	discoveryClient, err := restClientGetter(cfg.Client).ToDiscoveryClient()
	if err != nil {
		return err
	}
	if !cfg.Cached {
		discoveryClient.Invalidate()
	}

	lists, discoveryErr := discoveryClient.ServerPreferredResources()
	errList := []error{}
	if discoveryErr != nil {
		errList = append(errList, discoveryErr)
	}

	groupChanged := explicitFlagValue(cfg.APIGroupState)
	namespacedChanged := explicitFlagValue(cfg.NamespacedState)
	allResources := make([]v1.APIResource, 0)
	hasResourceList := false
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		groupVersion, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		hasResourceList = true
		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 || groupChanged && cfg.APIGroup != groupVersion.Group || namespacedChanged && cfg.Namespaced != resource.Namespaced {
				continue
			}
			if len(cfg.Verbs) > 0 && !sets.New[string](resource.Verbs...).HasAll(cfg.Verbs...) {
				continue
			}
			if len(cfg.Categories) > 0 && !sets.New[string](resource.Categories...).HasAll(cfg.Categories...) {
				continue
			}
			resource.Group = groupVersion.Group
			resource.Version = groupVersion.Version
			allResources = append(allResources, resource)
		}
	}
	if !hasResourceList {
		return errors.NewAggregate(errList)
	}

	slices.SortStableFunc(allResources, func(left, right v1.APIResource) int {
		leftKey, rightKey := left.Group, right.Group
		switch cfg.SortBy {
		case "name":
			leftKey, rightKey = left.Name, right.Name
		case "kind":
			leftKey, rightKey = left.Kind, right.Kind
		}
		if order := cmp.Compare(leftKey, rightKey); order != 0 {
			return order
		}
		return cmp.Compare(left.Name, right.Name)
	})

	flags := apiresources.NewPrintFlags()
	*flags.OutputFormat = cfg.Output
	*flags.NoHeaders = cfg.NoHeaders
	flags.JSONYamlPrintFlags.ShowManagedFields = cfg.ShowManagedFields
	printer, err := flags.ToPrinter()
	if err != nil {
		return err
	}
	list := &v1.APIResourceList{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "APIResourceList",
		},
		APIResources: allResources,
	}
	writer := printers.GetNewTabWriter(cfg.Stdout)
	defer writer.Flush()
	if !cfg.NoHeaders && (cfg.Output == "" || cfg.Output == "wide") {
		if err := printAPIResourceHeaders(writer, cfg.Output); err != nil {
			errList = append(errList, err)
		}
	}
	if err := printer.PrintObj(list, writer); err != nil {
		errList = append(errList, err)
	}
	return errors.NewAggregate(errList)
}

func printAPIResourceHeaders(out io.Writer, output string) error {
	columns := []string{"NAME", "SHORTNAMES", "APIVERSION", "NAMESPACED", "KIND"}
	if output == "wide" {
		columns = append(columns, "VERBS", "CATEGORIES")
	}
	_, err := fmt.Fprintf(out, "%s\n", strings.Join(columns, "\t"))
	return err
}

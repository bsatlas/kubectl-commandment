package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubectl/pkg/cmd/apiresources"
)

func runAPIResources(_ context.Context, cfg *apiResourcesConfig) error {
	if cfg.SortBy != "" && cfg.SortBy != "name" && cfg.SortBy != "kind" {
		return fmt.Errorf("--sort-by accepts only name or kind")
	}
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
	for _, list := range lists {
		groupVersion, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
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
	if len(allResources) == 0 {
		return errors.NewAggregate(errList)
	}

	sort.SliceStable(allResources, func(i, j int) bool {
		left, right := allResources[i].Group, allResources[j].Group
		switch cfg.SortBy {
		case "name":
			left, right = allResources[i].Name, allResources[j].Name
		case "kind":
			left, right = allResources[i].Kind, allResources[j].Kind
		case "":
		default:
			return false
		}
		if left == right {
			return allResources[i].Name < allResources[j].Name
		}
		return left < right
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

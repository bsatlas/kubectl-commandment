package main

import (
	"context"
	"slices"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/cmd/clusterinfo"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runClusterInfoDump(_ context.Context, cfg *clusterInfoDumpConfig) error {
	getter := restClientGetter(cfg.Client)
	printFlags := genericclioptions.NewPrintFlags("").WithTypeSetter(scheme.Scheme).WithDefaultOutput("json")
	configurePrintFlagsWithOutputState(printFlags, cfg.Output, cfg.Template, cfg.AllowMissingTemplateKeys, cfg.ShowManagedFields, cfg.OutputState)
	printer, err := printFlags.ToPrinter()
	if err != nil {
		return err
	}
	restConfig, err := getter.ToRESTConfig()
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	namespace, _, err := getter.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options := clusterinfo.ClusterInfoDumpOptions{
		PrintFlags:       printFlags,
		PrintObj:         printer.PrintObj,
		OutputDir:        cfg.OutputDirectory,
		AllNamespaces:    cfg.AllNamespaces,
		Namespaces:       slices.Clone(cfg.Namespaces),
		Timeout:          cfg.PodRunningTimeout,
		AppsClient:       clientset.AppsV1(),
		CoreClient:       clientset.CoreV1(),
		Namespace:        namespace,
		RESTClientGetter: getter,
		LogsForObject:    polymorphichelpers.LogsForObjectFn,
		IOStreams:        cfg.Streams,
	}
	return options.Run()
}

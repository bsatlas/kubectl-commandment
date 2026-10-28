package kubectl

import (
	"context"

	"k8s.io/kubectl/pkg/cmd/top"
	"k8s.io/kubectl/pkg/metricsutil"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

func runTopNode(_ context.Context, cfg *topNodeConfig) error {
	factory := cfg.Client.factory()
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	config, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	if cfg.UseProtocolBuffers {
		config.ContentType = "application/vnd.kubernetes.protobuf"
	}
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		return err
	}

	options := top.TopNodeOptions{
		ResourceName:    cfg.Name,
		Selector:        cfg.Selector,
		SortBy:          cfg.SortBy,
		NoHeaders:       cfg.NoHeaders,
		ShowCapacity:    cfg.ShowCapacity,
		ShowSwap:        cfg.ShowSwap,
		NodeClient:      clientset.CoreV1(),
		Printer:         metricsutil.NewTopCmdPrinter(cfg.Streams.Out, cfg.ShowSwap),
		DiscoveryClient: clientset.DiscoveryClient,
		MetricsClient:   metricsClient,
		IOStreams:       cfg.Streams,
	}
	return options.RunTopNode()
}

func runTopPod(_ context.Context, cfg *topPodConfig) error {
	factory := cfg.Client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	config, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	if cfg.UseProtocolBuffers {
		config.ContentType = "application/vnd.kubernetes.protobuf"
	}
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		return err
	}

	options := top.TopPodOptions{
		ResourceName:    cfg.Name,
		Namespace:       namespace,
		LabelSelector:   cfg.Selector,
		FieldSelector:   cfg.FieldSelector,
		SortBy:          cfg.SortBy,
		AllNamespaces:   cfg.AllNamespaces,
		PrintContainers: cfg.Containers,
		NoHeaders:       cfg.NoHeaders,
		Sum:             cfg.Sum,
		ShowSwap:        cfg.ShowSwap,
		PodClient:       clientset.CoreV1(),
		Printer:         metricsutil.NewTopCmdPrinter(cfg.Streams.Out, cfg.ShowSwap),
		DiscoveryClient: clientset.DiscoveryClient,
		MetricsClient:   metricsClient,
		IOStreams:       cfg.Streams,
	}
	return options.RunTopPod()
}

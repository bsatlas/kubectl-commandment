package main

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runExec(ctx context.Context, cfg *execConfig) error {
	factory := cfg.Client.factoryForInvocation(ctx)
	namespace, enforceNamespace, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	filenames := resource.FilenameOptions{
		Filenames: slices.Concat(cfg.Filename...),
	}
	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...)
	addFilenameSources(builder, enforceNamespace, filenames, cfg.Streams.In)
	builder.NamespaceParam(namespace).DefaultNamespace()
	if cfg.Resource != "" {
		builder.ResourceNames("pods", cfg.Resource)
	}
	object, err := builder.Do().Object()
	if err != nil {
		return err
	}
	if meta.IsListType(object) {
		return fmt.Errorf("cannot exec into multiple objects at a time")
	}
	pod, err := polymorphichelpers.AttachablePodForObjectFn(factory, object, cfg.PodRunningTimeout)
	if err != nil {
		return err
	}

	restConfig, err := factory.ToRESTConfig()
	if err != nil {
		return err
	}
	clientset, err := factory.KubernetesClientSet()
	if err != nil {
		return err
	}

	options := exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			Namespace:     namespace,
			PodName:       pod.Name,
			ContainerName: cfg.Container,
			Stdin:         cfg.Stdin,
			TTY:           cfg.TTY,
			Quiet:         cfg.Quiet,
			IOStreams:     cfg.Streams,
		},
		Command:       slices.Clone(cfg.Command),
		Executor:      &exec.DefaultRemoteExecutor{},
		PodClient:     clientset.CoreV1(),
		GetPodTimeout: cfg.PodRunningTimeout,
		Config:        restConfig,
	}
	return options.Run()
}

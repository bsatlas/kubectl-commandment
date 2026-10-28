package main

import (
	"context"
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/kubectl/pkg/cmd/logs"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runLogs(ctx context.Context, cfg *logsConfig) error {
	options := logs.NewLogsOptions(cfg.Streams)
	options.AllContainers = cfg.AllContainers
	options.AllPods = cfg.AllPods
	options.Container = cfg.Container
	options.ContainerNameSpecified = explicitFlagValue(cfg.ContainerState)
	options.Follow = cfg.Follow
	options.IgnoreLogErrors = cfg.IgnoreErrors
	options.InsecureSkipTLSVerifyBackend = cfg.InsecureSkipTLSVerifyBackend
	options.LimitBytes = cfg.LimitBytes
	options.MaxFollowConcurrency = cfg.MaxLogRequests
	options.Prefix = cfg.Prefix
	options.Previous = cfg.Previous
	options.Selector = cfg.Selector
	options.SinceSeconds = cfg.Since
	options.SinceTime = cfg.SinceTime
	options.Tail = cfg.Tail
	options.TailSpecified = explicitFlagValue(cfg.TailState)
	options.Timestamps = cfg.Timestamps
	options.GetPodTimeout = cfg.PodRunningTimeout
	options.Resources = append([]string(nil), cfg.Arguments...)

	switch len(cfg.Arguments) {
	case 0:
		if cfg.Selector == "" {
			return fmt.Errorf("POD or TYPE/NAME is a required argument for the logs command unless --selector is provided")
		}
	case 1:
		options.ResourceArg = cfg.Arguments[0]
		if cfg.Selector != "" {
			return fmt.Errorf("only a selector (-l) or a POD name is allowed")
		}
	case 2:
		options.ResourceArg = cfg.Arguments[0]
		options.Container = cfg.Arguments[1]
	default:
		return fmt.Errorf("logs accepts at most a resource and container")
	}
	if options.AllPods {
		options.Prefix = true
	}

	factory := cfg.Client.factory()
	namespace, _, err := factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}
	options.Namespace = namespace
	options.ConsumeRequestFn = logs.DefaultConsumeRequest
	options.RESTClientGetter = factory
	options.LogsForObject = polymorphichelpers.LogsForObjectFn
	options.AllPodLogsForObject = polymorphichelpers.AllPodLogsForObjectFn
	options.Options, err = options.ToLogOptions()
	if err != nil {
		return err
	}

	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace().
		SingleResourceType()
	if options.ResourceArg != "" {
		builder = builder.ResourceNames("pods", options.ResourceArg)
	}
	if options.Selector != "" {
		builder = builder.ResourceTypes("pods").LabelSelectorParam(options.Selector)
	}
	infos, err := builder.Do().Infos()
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("error from server (NotFound): %w in namespace %q", err, namespace)
		}
		return err
	}
	if options.Selector == "" && len(infos) != 1 {
		return fmt.Errorf("expected a resource")
	}
	options.Object = infos[0].Object
	if options.Selector != "" {
		pods, ok := options.Object.(*v1.PodList)
		if ok && len(pods.Items) == 0 {
			fmt.Fprintf(options.ErrOut, "No resources found in %s namespace.\n", namespace)
		}
	}

	if err := options.Validate(); err != nil {
		return err
	}
	return options.RunLogsContext(ctx)
}

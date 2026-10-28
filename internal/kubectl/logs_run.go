package kubectl

import (
	"context"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/kubectl/pkg/cmd/logs"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func runLogs(ctx context.Context, cfg *logsConfig) error {
	logOptions := &v1.PodLogOptions{
		Container:                    cfg.SelectedContainer,
		Follow:                       cfg.Follow,
		Previous:                     cfg.Previous,
		Timestamps:                   cfg.Timestamps,
		InsecureSkipTLSVerifyBackend: cfg.InsecureSkipTLSVerifyBackend,
	}
	if cfg.SinceTime != nil {
		sinceTime := metaNow()
		sinceTime.Time = *cfg.SinceTime
		logOptions.SinceTime = &sinceTime
	}
	if cfg.LimitBytes != 0 {
		limitBytes := cfg.LimitBytes
		logOptions.LimitBytes = &limitBytes
	}
	if cfg.Since != 0 {
		sinceSeconds := int64(cfg.Since.Round(time.Second).Seconds())
		logOptions.SinceSeconds = &sinceSeconds
	}
	if cfg.Tail != -1 {
		tail := cfg.Tail
		logOptions.TailLines = &tail
	}

	options := logs.NewLogsOptions(cfg.Streams)
	options.AllContainers = cfg.AllContainers
	options.AllPods = cfg.AllPods
	options.Follow = cfg.Follow
	options.IgnoreLogErrors = cfg.IgnoreErrors
	options.MaxFollowConcurrency = cfg.MaxLogRequests
	options.Prefix = cfg.EffectivePrefix
	options.Selector = cfg.Selector
	options.GetPodTimeout = cfg.PodRunningTimeout
	options.ResourceArg = cfg.Resource
	options.Options = logOptions

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

	builder := factory.NewBuilder().
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(namespace).DefaultNamespace().
		SingleResourceType()
	if cfg.Resource != "" {
		builder = builder.ResourceNames("pods", cfg.Resource)
	}
	if cfg.Selector != "" {
		builder = builder.ResourceTypes("pods").LabelSelectorParam(cfg.Selector)
	}
	infos, err := builder.Do().Infos()
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("error from server (NotFound): %w in namespace %q", err, namespace)
		}
		return err
	}
	if cfg.Selector == "" && len(infos) != 1 {
		return fmt.Errorf("expected a resource")
	}
	options.Object = infos[0].Object
	if cfg.Selector != "" {
		pods, ok := options.Object.(*v1.PodList)
		if ok && len(pods.Items) == 0 {
			fmt.Fprintf(options.ErrOut, "No resources found in %s namespace.\n", namespace)
		}
	}

	return options.RunLogsContext(ctx)
}

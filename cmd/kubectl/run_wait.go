package main

import (
	"context"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/watch"
	"k8s.io/kubectl/pkg/util/interrupt"
)

func waitForRunPod(ctx context.Context, clientset kubernetes.Interface, namespace, name string, timeout time.Duration, condition watch.ConditionFunc) (*v1.Pod, error) {
	ctx, cancel := watch.ContextWithOptionalTimeout(ctx, timeout)
	defer cancel()
	listWatch := runPodListWatch(clientset, namespace, name, ctx)
	intr := interrupt.New(nil, cancel)
	var result *v1.Pod
	err := intr.Run(func() error {
		event, err := watch.UntilWithSync(ctx, listWatch, &v1.Pod{}, nil, condition)
		if event != nil {
			result, _ = event.Object.(*v1.Pod)
		}
		return err
	})
	return result, err
}

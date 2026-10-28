package main

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"
)

func debugWatchContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return watch.ContextWithOptionalTimeout(ctx, timeout)
}

func debugUntilWithSync(ctx context.Context, listWatch *cache.ListWatch, object runtime.Object, condition watch.ConditionFunc) (runtime.Object, error) {
	event, err := watch.UntilWithSync(ctx, listWatch, object, nil, condition)
	if event == nil {
		return nil, err
	}
	return event.Object, err
}

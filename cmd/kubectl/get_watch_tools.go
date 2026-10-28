package main

import (
	"context"

	"k8s.io/client-go/tools/watch"
)

func runGetWatchUntil(ctx context.Context, stream getWatchStream, condition watch.ConditionFunc) error {
	_, err := watch.UntilWithoutRetry(ctx, stream, condition)
	return err
}

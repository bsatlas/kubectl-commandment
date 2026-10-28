package main

import "k8s.io/apimachinery/pkg/watch"

type getWatchStream interface {
	Stop()
	ResultChan() <-chan watch.Event
}

package main

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func parseServiceSelector(text string) (map[string]string, error) {
	selector, err := v1.ParseToLabelSelector(text)
	if err != nil {
		return nil, err
	}
	if len(selector.MatchExpressions) > 0 {
		return nil, fmt.Errorf("match expression %v not supported on this object", selector.MatchExpressions)
	}
	labels := make(map[string]string, len(selector.MatchLabels))
	for key, value := range selector.MatchLabels {
		labels[key] = value
	}
	return labels, nil
}

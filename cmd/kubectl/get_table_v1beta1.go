package main

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1beta1"
)

func getTableV1Beta1MediaType() string {
	return fmt.Sprintf("application/json;as=Table;v=%s;g=%s", v1beta1.SchemeGroupVersion.Version, v1beta1.GroupName)
}

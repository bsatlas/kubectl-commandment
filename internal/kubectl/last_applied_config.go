package kubectl

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/util"
)

func originalConfiguration(object runtime.Object) ([]byte, error) {
	return util.GetOriginalConfiguration(object)
}

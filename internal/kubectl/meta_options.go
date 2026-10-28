package kubectl

import "k8s.io/apimachinery/pkg/apis/meta/v1"

func getOptions() v1.GetOptions {
	return v1.GetOptions{}
}

func updateOptions() v1.UpdateOptions {
	return v1.UpdateOptions{}
}

func metaNow() v1.Time {
	return v1.Now()
}

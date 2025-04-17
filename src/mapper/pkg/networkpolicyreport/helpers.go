package networkpolicyreport

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func filterLargeAnnotations(annotations map[string]string) {
	for k, v := range annotations {
		if len(v) <= 100 {
			// Skip annotations with values longer than 100 characters
			continue
		}
		delete(annotations, k)
	}
}

func filterObjectMetadata(objectMeta metav1.ObjectMeta) metav1.ObjectMeta {
	objectMeta.Finalizers = nil
	objectMeta.ManagedFields = nil
	objectMeta.OwnerReferences = nil
	filterLargeAnnotations(objectMeta.Annotations)

	return objectMeta
}

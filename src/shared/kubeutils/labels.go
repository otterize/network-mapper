package kubeutils

func IsEnabledByLabel(labels map[string]string, labelKey string) bool {
	if labels == nil {
		return false
	}

	labelValue, ok := labels[labelKey]

	if !ok {
		return false
	}

	return labelValue == "true"
}

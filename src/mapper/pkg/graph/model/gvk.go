package model

import "k8s.io/apimachinery/pkg/runtime/schema"

func GroupVersionKindFromKubeGVK(kind schema.GroupVersionKind) *GroupVersionKind {
	gvk := &GroupVersionKind{
		Version: kind.Version,
		Kind:    kind.Kind,
	}

	if kind.Group != "" {
		gvk.Group = &kind.Group
	}

	return gvk
}

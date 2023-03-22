package model

import "k8s.io/apimachinery/pkg/types"

func (identity OtterizeServiceIdentity) AsNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      identity.Name,
		Namespace: identity.Namespace,
	}
}

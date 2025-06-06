package metadatareporter

import (
	"github.com/otterize/intents-operator/src/shared/serviceidresolver/serviceidentity"
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/nilable"
	"golang.org/x/exp/slices"
)

func labelsToLabelInput(labels map[string]string) []cloudclient.LabelInput {
	labelsInput := make([]cloudclient.LabelInput, 0)
	for key, value := range labels {
		labelsInput = append(labelsInput, cloudclient.LabelInput{Key: key, Value: nilable.From(value)})
	}

	slices.SortFunc(labelsInput, func(a, b cloudclient.LabelInput) int {
		if a.Key < b.Key {
			return -1
		}
		if a.Key > b.Key {
			return 1
		}
		if !a.Value.Set && !b.Value.Set {
			return 0
		}
		if !a.Value.Set {
			return -1
		}
		if !b.Value.Set {
			return 1
		}
		if a.Value.Item < b.Value.Item {
			return -1
		}
		if a.Value.Item > b.Value.Item {
			return 1
		}
		return 0
	})
	return labelsInput
}

func serviceIdentityToServiceIdentityInput(identity serviceidentity.ServiceIdentity) cloudclient.ServiceIdentityInput {
	wi := cloudclient.ServiceIdentityInput{
		Namespace: identity.Namespace,
		Name:      identity.Name,
		Kind:      identity.Kind,
	}
	if identity.ResolvedUsingOverrideAnnotation != nil {
		wi.NameResolvedUsingAnnotation = nilable.From(*identity.ResolvedUsingOverrideAnnotation)
	}

	return wi
}

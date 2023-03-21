package clouduploader

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
)

func modelKafkaOpToAPI(op model.KafkaOperation) cloudclient.KafkaOperation {
	return cloudclient.KafkaOperation(op)
}

func modelKafkaConfToAPI(kc model.KafkaConfig) cloudclient.KafkaConfigInput {
	return cloudclient.KafkaConfigInput{
		Name: lo.ToPtr(kc.Name),
		Operations: lo.Map(kc.Operations, func(op model.KafkaOperation, _ int) *cloudclient.KafkaOperation {
			return lo.ToPtr(modelKafkaOpToAPI(op))
		}),
	}
}

func modelIntentTypeToAPI(it *model.IntentType) *cloudclient.IntentType {
	if it == nil {
		return nil
	}
	return lo.ToPtr(cloudclient.IntentType(lo.FromPtr(it)))
}

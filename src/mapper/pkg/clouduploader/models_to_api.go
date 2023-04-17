package clouduploader

import (
	"github.com/otterize/network-mapper/src/mapper/pkg/cloudclient"
	"github.com/otterize/network-mapper/src/mapper/pkg/graph/model"
	"github.com/samber/lo"
)

var modelMethodToAPIMethodMap = map[model.HTTPMethod]cloudclient.HTTPMethod{
	model.HTTPMethodGet:     cloudclient.HTTPMethodGet,
	model.HTTPMethodPost:    cloudclient.HTTPMethodPost,
	model.HTTPMethodPut:     cloudclient.HTTPMethodPut,
	model.HTTPMethodPatch:   cloudclient.HTTPMethodPatch,
	model.HTTPMethodDelete:  cloudclient.HTTPMethodDelete,
	model.HTTPMethodConnect: cloudclient.HTTPMethodConnect,
	model.HTTPMethodOptions: cloudclient.HTTPMethodOptions,
	model.HTTPMethodTrace:   cloudclient.HTTPMethodTrace,
	model.HTTPMethodAll:     cloudclient.HTTPMethodAll,
}

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

func modelHTTPMethodToAPI(method model.HTTPMethod) cloudclient.HTTPMethod {
	return modelMethodToAPIMethodMap[method]
}

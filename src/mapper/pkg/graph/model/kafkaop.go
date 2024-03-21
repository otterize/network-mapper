package model

import (
	"github.com/otterize/intents-operator/src/shared/errors"
	"strings"
)

var (
	kafkaOperationToAclOperation = map[string]KafkaOperation{
		"read":            KafkaOperationConsume,
		"write":           KafkaOperationProduce,
		"create":          KafkaOperationCreate,
		"delete":          KafkaOperationDelete,
		"alter":           KafkaOperationAlter,
		"describe":        KafkaOperationDescribe,
		"clusteraction":   KafkaOperationClusterAction,
		"describeconfigs": KafkaOperationDescribeConfigs,
		"alterconfigs":    KafkaOperationAlterConfigs,
		"idempotentwrite": KafkaOperationIdempotentWrite,
	}
)

func KafkaOpFromText(text string) (KafkaOperation, error) {
	normalized := strings.ToLower(text)

	apiOp, ok := kafkaOperationToAclOperation[normalized]
	if !ok {
		return "", errors.Errorf("failed parsing op %s", text)
	}
	return apiOp, nil
}

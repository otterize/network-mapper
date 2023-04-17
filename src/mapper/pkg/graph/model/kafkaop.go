package model

import (
	"fmt"
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
		"idempotentwrite": KafkaOperationIDEmpotentWrite,
	}
)

func KafkaOpFromText(text string) (KafkaOperation, error) {
	normalized := strings.ToLower(text)

	apiOp, ok := kafkaOperationToAclOperation[normalized]
	if !ok {
		return "", fmt.Errorf("failed parsing op %s", text)
	}
	return apiOp, nil
}

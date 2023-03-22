package model

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/vishalkuo/bimap"
)

var (
	kafkaOperationToAclOperation = map[KafkaOperation]sarama.AclOperation{
		KafkaOperationAll:             sarama.AclOperationAll,
		KafkaOperationConsume:         sarama.AclOperationRead,
		KafkaOperationProduce:         sarama.AclOperationWrite,
		KafkaOperationCreate:          sarama.AclOperationCreate,
		KafkaOperationDelete:          sarama.AclOperationDelete,
		KafkaOperationAlter:           sarama.AclOperationAlter,
		KafkaOperationDescribe:        sarama.AclOperationDescribe,
		KafkaOperationClusterAction:   sarama.AclOperationClusterAction,
		KafkaOperationDescribeConfigs: sarama.AclOperationDescribeConfigs,
		KafkaOperationAlterConfigs:    sarama.AclOperationAlterConfigs,
		KafkaOperationIDEmpotentWrite: sarama.AclOperationIdempotentWrite,
	}
	KafkaOperationToAclOperationBMap = bimap.NewBiMapFromMap(kafkaOperationToAclOperation)
)

func KafkaOpFromText(text string) (KafkaOperation, error) {
	var saramaOp sarama.AclOperation
	if err := saramaOp.UnmarshalText([]byte(text)); err != nil {
		return "", err
	}

	apiOp, ok := KafkaOperationToAclOperationBMap.GetInverse(saramaOp)
	if !ok {
		return "", fmt.Errorf("failed parsing op %s", saramaOp.String())
	}
	return apiOp, nil
}

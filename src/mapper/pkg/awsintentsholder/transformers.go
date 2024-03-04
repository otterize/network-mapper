package awsintentsholder

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"strings"
)

type AWSIntentsTransformer interface {
	Transform(intents []AWSIntent) []AWSIntent
}

type AWSIntentRootTransformer struct{}
type AWSIntentFlattenTransformer struct{}
type S3ObjectTransformer struct{}

func (g *AWSIntentRootTransformer) Transform(intents []AWSIntent) []AWSIntent {
	transformers := []AWSIntentsTransformer{
		&AWSIntentFlattenTransformer{},
		&S3ObjectTransformer{},
	}

	for _, transformer := range transformers {
		intents = transformer.Transform(intents)
	}

	return intents

}
func (g *AWSIntentFlattenTransformer) Transform(intents []AWSIntent) []AWSIntent {
	result := make([]AWSIntent, 0)

	for _, intent := range intents {
		for _, action := range intent.Actions {
			result = append(result, AWSIntent{
				Client:  intent.Client,
				Actions: []string{action},
				ARN:     intent.ARN,
			})
		}
	}

	return result
}

func (g *S3ObjectTransformer) Transform(intents []AWSIntent) []AWSIntent {
	result := make([]AWSIntent, 0)

	for _, intent := range intents {
		arnId, err := arn.Parse(intent.ARN)

		if err != nil {
			logrus.WithError(err).Warnf("Failed to parse ARN: %s", intent.ARN)
			continue
		}

		isS3Object, bucketWildCardArn := g.isS3Object(arnId)

		if isS3Object {
			intent.ARN = bucketWildCardArn.String()
		}

		result = append(result, intent)
	}

	return result
}

func (g *S3ObjectTransformer) isS3Object(arnId arn.ARN) (bool, *arn.ARN) {
	if arnId.Service != "s3" {
		return false, nil
	}

	resource := arnId.Resource

	if resource == "" {
		return false, nil
	}

	before, _, found := strings.Cut(resource, "/")

	if !found {
		return false, nil
	}

	reservedNames := []string{
		"accesspoint",
		"job",
		"storage-lens",
		"storage-lens-group",
		"async-request",
		"access-grants",
	}

	// skip non-object resources
	if lo.Contains(reservedNames, before) {
		return false, nil
	}

	bucketName := before
	bucketWildCardArn := arnId
	bucketWildCardArn.Resource = fmt.Sprintf("%s/*", bucketName)
	return true, &bucketWildCardArn
}

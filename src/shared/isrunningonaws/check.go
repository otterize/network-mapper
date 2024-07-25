package isrunningonaws

import (
	"context"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/smithy-go/logging"
	"github.com/sirupsen/logrus"
	"time"
)

func Check() bool {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cfg, err := awsconfig.LoadDefaultConfig(ctxTimeout)
	if err != nil {
		logrus.Debug("Autodetect AWS (an error here is fine): Failed to load AWS config")
		return false
	}
	cfg.Logger = logging.Nop{}

	client := imds.NewFromConfig(cfg)

	result, err := client.GetInstanceIdentityDocument(ctxTimeout, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		logrus.Debug("Autodetect AWS (an error here is fine): Failed to get instance identity document")
		return false
	}

	logrus.WithField("region", result.Region).Debug("Autodetect AWS: Running on AWS")
	return true
}

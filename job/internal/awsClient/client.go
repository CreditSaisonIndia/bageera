package awsClient

import (
	"context"
	"os"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
)

var LOGGER *zap.SugaredLogger

func getSession() (*session.Session, error) {
	// AWS session setup
	LOGGER = customLogger.GetLogger()
	var region string
	if serviceConfig.ApplicationSetting.RunType == "local" {
		LOGGER.Info("Using Local ini for QUEUE URL")
		region = serviceConfig.ApplicationSetting.Region
	} else {
		region = os.Getenv("AWS_REGION")
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		// Add other AWS configurations as needed
	})
	if err != nil {
		LOGGER.Info("Error creating AWS session:", err)
		return nil, err
	}
	return sess, nil
}

func GetSqsClient() (*sqs.SQS, error) {
	// SQS client
	sess, err := getSession()
	if err != nil {
		LOGGER.Info("Error creating AWS session:", err)
		return nil, err
	}
	return sqs.New(sess), nil
}

func GetSecretValue(client *secretsmanager.Client, secretNameOrARN string) (string, error) {
	LOGGER = customLogger.GetLogger()
	LOGGER.Info("*****GETTING SECRETS*****")
	// Create input for the GetSecretValue operation
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretNameOrARN),
	}

	// Send request to AWS Secrets Manager
	result, err := client.GetSecretValue(context.TODO(), input)
	if err != nil {
		return "", err
	}

	// Check if the secret has a secret string
	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	// If the secret has a binary secret, you would handle it here.
	// Note: In this example, we only handle the case of a secret string.

	return "", err
}

package awsClient

import (
	"context"
	"fmt"

	"encoding/json"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
)

var LOGGER *zap.SugaredLogger

func getSession() (*session.Session, error) {
	// AWS session setup
	LOGGER = customLogger.GetLogger()
	region := serviceConfig.ApplicationSetting.Region

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

func getSnsClient() (*sns.SNS, error) {
	sess, err := getSession()
	if err != nil {
		LOGGER.Info("Error creating AWS session:", err)
		return nil, err
	}
	return sns.New(sess), nil
}

func Publish(baseAlert model.BaseAlert, snsArn string) {

	LOGGER.Info("*** SENDING ALERT ***")
	snsClient, err := getSnsClient()
	if err != nil {
		LOGGER.Error("Error creating SNS client | Error", err)
		LOGGER.Info("SKIPPING ALERT")
		return
	}

	// Marshal the struct to a JSON string
	jsonData, err := json.Marshal(baseAlert)
	if err != nil {
		LOGGER.Error("Error marshaling baseAlert to json:", err)
		LOGGER.Info("SKIPPING ALERT")
		return
	}
	alertMessageStr := string(jsonData)
	// Publish the message
	result, err := snsClient.Publish(&sns.PublishInput{
		Message:   aws.String(alertMessageStr),
		TargetArn: aws.String(snsArn),
	})
	if err != nil {
		fmt.Println("Error publishing message:", err)
		LOGGER.Info("SKIPPING ALERT")
		return
	}
	LOGGER.Info("MESSAGE : ", alertMessageStr)
	LOGGER.Info("Message sent, MessageID:", *result.MessageId)
	LOGGER.Info("*** ALERT SENT ***")

}

func GetS3Client() (*s3.S3, error) {

	sess, err := getSession()
	if err != nil {
		LOGGER.Info("Error creating AWS session:", err)
		return nil, err
	}
	return s3.New(sess), nil
}

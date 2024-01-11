package queueConsumer

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient"
	"github.com/CreditSaisonIndia/bageera/internal/consolidation"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/splitter"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type S3UploadEvent struct {
	Event           string `json:"event"`
	BucketName      string `json:"bucketName"`
	ObjectKey       string `json:"objectKey"`
	LPC             string `json:"lpc"`
	FileName        string `json:"fileName"`
	Execution       string `json:"execution"`
	Region          string `json:"region"`
	RequestQueueUrl string `json:"requestQueueUrl"`
	DBUserName      string `json:"dbUserName"`
	DBPassword      string `json:"dbPassword"`
	DBHost          string `json:"dbHost"`
	DBPort          string `json:"dbPort"`
	DBName          string `json:"dbName"`
	Schema          string `json:"schema"`
	EFSBasePath     string `json:"efsBasePath"`
	Environment     string `json:"environment"`
}
type SNSNotification struct {
	Type             string `json:"Type"`
	MessageId        string `json:"MessageId"`
	TopicArn         string `json:"TopicArn"`
	Message          string `json:"Message"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
}

type SecretData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Consume() error {
	LOGGER := customLogger.GetLogger()

	// SQS client
	sqsClient, err := awsClient.GetSqsClient()
	if err != nil {
		log.Println("Error creating AWS session:", err)
		return err
	}
	var queueURL string
	if serviceConfig.ApplicationSetting.RunType == "local" {

		queueURL = serviceConfig.ApplicationSetting.PqJobQueueUrl

		LOGGER.Info("Using Local ini for QUEUE URL | ", queueURL)
	} else {
		queueURL = os.Getenv("requestQueueUrl")
	}

	// Specify your queue URL

	// Run the polling loop until there are no more messages
	for {
		LOGGER.Info("Listening for messages ...")
		time.Sleep(5 * time.Second)
		// Poll for messages
		receiveParams := &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: aws.Int64(1),
			VisibilityTimeout:   aws.Int64(10), // Adjust visibility timeout as needed
			WaitTimeSeconds:     aws.Int64(20), // Adjust wait time as needed
		}

		result, err := sqsClient.ReceiveMessage(receiveParams)
		if err != nil {
			LOGGER.Info("Error receiving message:", err)
			break // Exit the loop on error
		}

		// Check if there are no more messages
		if len(result.Messages) == 0 {
			LOGGER.Info("No more messages in the queue. Exiting...")
			//close(producer.ProducerConcurrencyCh)
			//close(consumer.ConsumerConcurrencyCh)
			break
		}

		// Process the received messages
		for _, msg := range result.Messages {
			// Handle the message as needed
			LOGGER.Info("Received message:", *msg.Body)
			setConfigFromSqsMessage(*msg.Body)
			startTime := time.Now()

			log.Printf("*********BEGIN********")

			//logFile, err := fileUtilityWrapper.AddLogFileSugar()
			if err != nil {
				LOGGER.Info("Error while creating log file : ", err)
				return err
			}
			chunksDir := utils.GetChunksDir()
			fileUtilityWrapper.DeleteDirIfExist(chunksDir)
			logFile, err := fileUtilityWrapper.AddLogFile()
			path := fileUtilityWrapper.S3FileDownload()
			LOGGER.Info("downloaded file path:", path)
			splitter.SplitCsv()
			insertion.BeginInsertion()

			consolidation.Consolidate()

			LOGGER.Info("Exiting...")
			endTime := time.Now()
			elapsedTime := endTime.Sub(startTime)
			elapsedMinutes := elapsedTime.Minutes()
			log.Printf("Time taken: %.2f minutes\n", elapsedMinutes)
			func(logFile *os.File) {
				err := logFile.Close()
				if err != nil {
					LOGGER.Info("Error While closing the log file")
				}
			}(logFile)

			// Delete the message from the queue after processing
			deleteParams := &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: msg.ReceiptHandle,
			}
			_, err = sqsClient.DeleteMessage(deleteParams)
			if err != nil {
				LOGGER.Info("Error deleting message:", err)
			}
		}
	}
	return nil
}

func setConfigFromSqsMessage(jsonMessage string) {
	LOGGER := customLogger.GetLogger()
	//LOGGER.Info("GETTING SECRETS")
	//
	//// Load AWS SDK configuration
	//cfg, err := config.LoadDefaultConfig(context.TODO())
	//if err != nil {
	//	LOGGER.Info("Error loading AWS SDK configuration:", err)
	//	return
	//}
	//
	//// Create a new Secrets Manager client
	//client := secretsmanager.NewFromConfig(cfg)
	//
	//// Specify the ARN or name of your secret
	//secretNameOrARN := "scarlet-db-creds"
	//
	//// Retrieve the secret value
	//secretValue, err := awsClient.GetSecretValue(client, secretNameOrARN)
	//if err != nil {
	//	LOGGER.Info("Error retrieving secret value:", err)
	//	return
	//}
	//
	//// Unmarshal the JSON into a struct
	//var secretData SecretData
	//err = json.Unmarshal([]byte(secretValue), &secretData)
	//if err != nil {
	//	LOGGER.Info("Error unmarshaling secret data:", err)
	//	return
	//}
	//
	//LOGGER.Info("GOT SECRETS SUCCESSFULLY")

	LOGGER.Info("SETTING DATA FROM QUEUE MESSAGE")
	// Unmarshal the outer SNSNotification
	var snsNotification SNSNotification
	err := json.Unmarshal([]byte(jsonMessage), &snsNotification)
	if err != nil {
		LOGGER.Info("Error decoding JSON:", err)
		return
	}

	// Unmarshal the inner S3UploadEvent
	var s3UploadEvent S3UploadEvent
	err = json.Unmarshal([]byte(snsNotification.Message), &s3UploadEvent)
	if err != nil {
		LOGGER.Info("Error decoding inner JSON:", err)
		return
	}

	// Extracted values

	fileName := s3UploadEvent.FileName
	serviceConfig.Set("fileName", fileName)

	bucketName := s3UploadEvent.BucketName
	serviceConfig.Set("bucketName", bucketName)

	objectKey := s3UploadEvent.ObjectKey
	serviceConfig.Set("objectKey", objectKey)

	lpc := s3UploadEvent.LPC
	serviceConfig.Set("lpc", lpc)

	execution := s3UploadEvent.Execution
	serviceConfig.Set("execution", execution)

	region := s3UploadEvent.Region
	serviceConfig.Set("region", region)

	requestQueueUrl := s3UploadEvent.RequestQueueUrl
	serviceConfig.Set("requestQueueUrl", requestQueueUrl)

	//dbUsername := s3UploadEvent.DBUserName
	serviceConfig.Set("dbUsername", serviceConfig.DatabaseSetting.User)

	//dbPassword := s3UploadEvent.DBPassword
	serviceConfig.Set("dbPassword", serviceConfig.DatabaseSetting.Password)

	//dbHost := s3UploadEvent.DBHost
	dbHost := serviceConfig.DatabaseSetting.MasterDbHost
	serviceConfig.Set("dbHost", dbHost)

	dbPort := serviceConfig.DatabaseSetting.Port
	serviceConfig.Set("dbPort", dbPort)

	dbName := serviceConfig.DatabaseSetting.Name
	serviceConfig.Set("dbName", dbName)

	schema := serviceConfig.DatabaseSetting.TablePrefix
	serviceConfig.Set("schema", schema)

	efsBasePath := serviceConfig.ApplicationSetting.EfsBasePath
	serviceConfig.Set("efsBasePath", efsBasePath)

	environment := s3UploadEvent.Environment
	serviceConfig.Set("environment", environment)

	LOGGER.Info("DATA FROM QUEUE MESSAGE PARSED SUCCESSFULLY ")

}

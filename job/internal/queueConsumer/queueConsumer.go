package queueConsumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient"
	"github.com/CreditSaisonIndia/bageera/internal/consolidation"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/database"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/splitter"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"github.com/CreditSaisonIndia/bageera/internal/validation"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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
			err = setConfigFromSqsMessage(*msg.Body)
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error(err)
				break
			}
			deleteParams := &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: msg.ReceiptHandle,
			}
			startTime := time.Now()

			LOGGER.Info("*********BEGIN********")
			sendAlertMessage("IN_PROGRESS", "")
			//logFile, err := fileUtilityWrapper.AddLogFileSugar()
			// if err != nil {
			// 	LOGGER.Info("Error while creating log file : ", err)
			// 	return err
			// }
			chunksDir := utils.GetChunksDir()
			err = fileUtilityWrapper.DeleteDirIfExist(chunksDir)
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error(err)
				break
			}
			err := os.MkdirAll(chunksDir, os.ModePerm)
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error(err)
				break
			}
			logFile, err := fileUtilityWrapper.AddLogFile()
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error(err)
				break
			}
			path, err := fileUtilityWrapper.S3FileDownload()
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error(err)
				if err.Error() == "S3KeyError" {
					// Alert sent in the internal function already
					delteMessageFromSQS(deleteParams, sqsClient)
				}
				break
			}

			LOGGER.Info("downloaded file path:", path)

			err = delteMessageFromSQS(deleteParams, sqsClient)
			if err != nil {
				LOGGER.Error("Error while deleting the message from queue", err)
				sendAlertMessage("ERROR", fmt.Sprintf("Error while deleting the message from queue - %s", err))
				break
			}

			LOGGER.Info("Validating the csv file at path:", path)
			anyValidRow, err := validation.Validate(path)
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error(err)
				sendAlertMessage("FAILED", fmt.Sprintf("Failed while validating the CSV | Remarks - %s", err))
				break
			}
			invalidGoroutinesWaitGroup := sync.WaitGroup{}

			uploadInvalidFileToS3IfExist(&invalidGoroutinesWaitGroup)

			if !anyValidRow {
				LOGGER.Error("No valid rows present after validation", anyValidRow)
				sendAlertMessage("FAILED", "No Valid rows present after validation")
				LOGGER.Info("Starting initializeWait")
				initializeWait(&invalidGoroutinesWaitGroup)
				LOGGER.Info("Ended initializeWait")
				break
			}

			LOGGER.Info("Splitting...")
			err = splitter.SplitCsv()
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error("ERROR WHILE SPLITTING CSV : ", err)
				sendAlertMessage("FAILED", fmt.Sprintf("ERROR WHILE SPLITTING CSV - %s", err))
				break
			}

			// Initialize the global CustomDBManager from the new package
			// database.InitSqlxDb()
			LOGGER.Info("SETTING SESSION FOR DATABASE")
			opts := session.Options{Config: aws.Config{
				CredentialsChainVerboseErrors: aws.Bool(true),
				Region:                        aws.String(serviceConfig.ApplicationSetting.Region),
				MaxRetries:                    aws.Int(3),
			}}
			sess := session.Must(session.NewSessionWithOptions(opts))

			LOGGER.Info("DONE SETTING SESSION FOR DATABASE")
			// Create a peer instance
			p := &database.Peer{
				Name:        "peer",
				Logger:      customLogger.GetLogger(), // Adjust the logger as needed
				IAMRoleAuth: true,                     // Set to true if you want to use IAM role authentication
			}

			// Define your database configuration
			cfg := database.DBConfig{
				Host:        serviceConfig.DatabaseSetting.MasterDbHost,
				Port:        5432, // Adjust the port as needed
				User:        serviceConfig.DatabaseSetting.User,
				Password:    serviceConfig.DatabaseSetting.Password,
				SSLMode:     "require", // Adjust the SSL mode as needed
				Name:        serviceConfig.DatabaseSetting.Name,
				MinConn:     5,
				MaxConn:     20,
				LifeTime:    "14m",
				IdleTime:    "5m",
				LogLevel:    "info", // Adjust the log level as needed
				Region:      os.Getenv("region"),
				IAMRoleAuth: true, // Set to true if you want to use IAM role authentication
			}

			// Get a database connection pool
			pool, err := p.GetDBPool(context.Background(), cfg, sess)
			if err != nil {
				LOGGER.Error("ERROR WHILE INITIALIZING DB POOL : ", err)
				serviceConfig.PrintSettings()
				sendAlertMessage("FAILED", fmt.Sprintf("ERROR WHILE INITIALIZING DB POOL - %s", err))
				break
			}

			// Close the pool when you're done with it
			defer pool.Close()
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
					LOGGER.Error("Error While closing the log file: ", err)
					serviceConfig.PrintSettings()

				}
			}(logFile)

			baseAlert := model.BaseAlert{
				FileName: serviceConfig.ApplicationSetting.FileName,
				Lpc:      serviceConfig.ApplicationSetting.Lpc,
				Status:   "SUCCESS",
			}
			awsClient.Publish(baseAlert, serviceConfig.ApplicationSetting.AlertSnsArn)
			LOGGER.Info("**********CLOSING POOL************")
			pool.Close()
		}
	}
	return nil
}

func uploadInvalidFileToS3IfExist(invalidGoroutinesWaitGroup *sync.WaitGroup) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("*******UPLOADING INVALID FILE*******")

	invalidGoroutinesWaitGroup.Add(1)

	go awsClient.S3MutiPartUpload()
}

func setConfigFromSqsMessage(jsonMessage string) error {
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
		LOGGER.Error("Error decoding JSON:", err)
		return err
	}

	// Unmarshal the inner S3UploadEvent
	var s3UploadEvent S3UploadEvent
	err = json.Unmarshal([]byte(snsNotification.Message), &s3UploadEvent)
	if err != nil {
		LOGGER.Error("Error decoding inner JSON:", err)
		return err
	}

	// Extracted values
	serviceConfig.ApplicationSetting.EfsBasePath = os.Getenv("efsBathPath")
	serviceConfig.ApplicationSetting.Region = os.Getenv("region")
	fileName := s3UploadEvent.FileName
	serviceConfig.ApplicationSetting.FileName = fileName
	serviceConfig.Set("fileName", fileName)

	bucketName := s3UploadEvent.BucketName
	serviceConfig.ApplicationSetting.BucketName = bucketName
	serviceConfig.Set("bucketName", bucketName)

	objectKey := s3UploadEvent.ObjectKey
	serviceConfig.ApplicationSetting.ObjectKey = objectKey
	serviceConfig.Set("objectKey", objectKey)

	invalidObjectKey := utils.GetInvalidObjectKey()
	serviceConfig.ApplicationSetting.InvalidObjectKey = invalidObjectKey

	lpc := s3UploadEvent.LPC
	serviceConfig.ApplicationSetting.Lpc = lpc
	serviceConfig.Set("lpc", lpc)

	// execution := s3UploadEvent.Execution
	// serviceConfig.Set("execution", execution)

	// requestQueueUrl := s3UploadEvent.RequestQueueUrl
	// serviceConfig.ApplicationSetting.PqJobQueueUrl = requestQueueUrl
	// serviceConfig.Set("requestQueueUrl", requestQueueUrl)

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

	return nil

}

func delteMessageFromSQS(deleteParams *sqs.DeleteMessageInput, sqsClient *sqs.SQS) error {
	LOGGER := customLogger.GetLogger()
	_, err := sqsClient.DeleteMessage(deleteParams)
	if err != nil {
		LOGGER.Error("Error deleting message:", err)
		serviceConfig.PrintSettings()
		return err
	}
	return nil
}

func sendAlertMessage(status string, message string) {
	baseAlert := model.BaseAlert{
		FileName: serviceConfig.ApplicationSetting.FileName,
		Lpc:      serviceConfig.ApplicationSetting.Lpc,
		Status:   status,
		Message:  message,
	}
	awsClient.Publish(baseAlert, serviceConfig.ApplicationSetting.AlertSnsArn)
}

func initializeWait(invalidGoroutinesWaitGroup *sync.WaitGroup) {
	LOGGER := customLogger.GetLogger()
	go func() {
		invalidGoroutinesWaitGroup.Wait()
		LOGGER.Info("*******INVALID FILE UPLOAD CALL DONE*******")
	}()
}

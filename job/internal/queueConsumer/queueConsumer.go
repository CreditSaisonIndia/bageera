package queueConsumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient"
	"github.com/CreditSaisonIndia/bageera/internal/awsClient/multipartUpload"
	"github.com/CreditSaisonIndia/bageera/internal/consolidation"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/database"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/job"
	"github.com/CreditSaisonIndia/bageera/internal/job/existence"
	"github.com/CreditSaisonIndia/bageera/internal/sequentialValidator"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
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
			awsClient.SendAlertMessage("IN_PROGRESS", "")
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
				awsClient.SendAlertMessage("ERROR", fmt.Sprintf("Error while deleting the message from queue - %s", err))
				break
			}

			LOGGER.Info("Validating the csv file at path:", path)
			allInvalidRows, err := sequentialValidator.Validate(path)
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error("Error while Validation ", err)
				awsClient.SendAlertMessage("FAILED", fmt.Sprintf("Failed while validating the CSV | Remarks - %s", err))
				break
			}

			if allInvalidRows {
				serviceConfig.PrintSettings()
				LOGGER.Info("Starting invalid upload file Wait")
				invalidGoroutinesWaitGroup := sync.WaitGroup{}
				invalidGoroutinesWaitGroup.Add(1)
				invalidBaseDir := utils.GetInvalidBaseDir()
				fileNameWithoutExt, _ := utils.GetFileName()
				uploadInvalidFileToS3IfExist(&invalidGoroutinesWaitGroup,
					filepath.Join(invalidBaseDir, fileNameWithoutExt+"_invalid.csv"), utils.GetRelativeInvalidBaseDir())
				LOGGER.Info("*******INVALID FILE UPLOAD CALL DONE*******")
				LOGGER.Info("Ended invalidGoroutinesWaitGroup Wait")
				break
			}

			existence := &existence.Existence{
				LOGGER: customLogger.GetLogger(), // Adjust the logger as needed
				// Set to true if you want to use IAM role authentication
			}

			err = existence.ExecuteJob(serviceConfig.ApplicationSetting.ObjectKey, "initial_offer")
			if err != nil {
				LOGGER.Error("ERROR WHILE existence.Execute() : ", err)
				break
			}
			fileNameWithoutExt, _ := utils.GetFileName()
			fileNameWithoutExt += "_valid"
			failureFilePathFormat := "%s_%s_exist_failure.csv"
			successFilePathFormat := "%s_%s_exist_success.csv"
			failurePattern := regexp.MustCompile(fmt.Sprintf(`^%s_\d+_exist_failure\.csv$`, fileNameWithoutExt))
			successPattern := regexp.MustCompile(fmt.Sprintf(`^%s_\d+_exist_success\.csv$`, fileNameWithoutExt))

			existRowCountFilePath, err := consolidation.Consolidate(failurePattern, successPattern, failureFilePathFormat, successFilePathFormat, fileNameWithoutExt, "exist_row_count.csv", 3)
			if err != nil {

				//UPLOADING INVALID FILE
				LOGGER.Error("ERROR WHILE CONSOLIDATION : ", err)
				serviceConfig.PrintSettings()
				awsClient.SendAlertMessage("FAILED", fmt.Sprintf("ERROR WHILE CONSOLIDATION - %s", err))
				LOGGER.Info("Starting invalid upload file Wait")
				invalidGoroutinesWaitGroup := sync.WaitGroup{}
				invalidGoroutinesWaitGroup.Add(1)
				invalidBaseDir := utils.GetInvalidBaseDir()
				fileNameWithoutExt, _ := utils.GetFileName()
				uploadInvalidFileToS3IfExist(&invalidGoroutinesWaitGroup,
					filepath.Join(invalidBaseDir, fileNameWithoutExt+"_invalid.csv"), utils.GetRelativeInvalidBaseDir())
				LOGGER.Info("*******INVALID FILE UPLOAD CALL DONE*******")
				LOGGER.Info("Ended invalidGoroutinesWaitGroup Wait")

				break
			}

			verifier := &consolidation.ExistenceVerifyConsolidatorImpl{}
			result := verifier.VerifyCount(existRowCountFilePath)
			if result.SomePresent {
				LOGGER.Info("******CSV HAS SOME NEW OFFERS TO DUMP******")
				if serviceConfig.ApplicationSetting.JobType == "delete" {
					LOGGER.Info("******JOB IS OF TYPE DELETE HENCE DUMPING OFFERS TO HISTORY TABLE******")
					err = existence.DoInsert()
				}

			} else if result.AllPresent {
				LOGGER.Error("******CSV HAS NO NEW OFFERS TO DUMP******")
				//not breaking the flow if the job type is of delete
				if serviceConfig.ApplicationSetting.JobType == "insert" {
					awsClient.SendAlertMessage("FAILED", "All Offers are Pre-Existing")
					break
				}

				if serviceConfig.ApplicationSetting.JobType == "delete" {
					LOGGER.Info("******JOB IS OF TYPE DELETE HENCE DUMPING OFFERS TO HISTORY TABLE******")
					err = existence.DoInsert()
				}

				LOGGER.Error("******SKIPPING THE BREAK SINCE JOB IS OF TYPE DELETE/UPDATE******")

			} else if result.AllAbsent {
				LOGGER.Error("******CSV HAS WHOLE NEW OFFERS TO DUMP******")
				//not breaking the flow if the job type is of delete
				if serviceConfig.ApplicationSetting.JobType == "update" || serviceConfig.ApplicationSetting.JobType == "delete" {

					LOGGER.Error("******TERMINATING THE JOB SINCE ITS OF TYPE DELETE/UPDATE******")
					awsClient.SendAlertMessage("FAILED", "All Offers are Non-Existing")

					break
				}
				LOGGER.Error("******SKIPPING THE BREAK SINCE JOB IS OF TYPE INSERT******")
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
				Port:        serviceConfig.DatabaseSetting.Port, // Adjust the port as needed
				User:        serviceConfig.DatabaseSetting.User,
				Password:    serviceConfig.DatabaseSetting.Password,
				SSLMode:     serviceConfig.DatabaseSetting.SslMode, // Adjust the SSL mode as needed
				Name:        serviceConfig.DatabaseSetting.Name,
				Region:      os.Getenv("region"),
				IAMRoleAuth: true, // Set to true if you want to use IAM role authentication
				Env:         os.Getenv("environment"),
				SearchPath:  serviceConfig.DatabaseSetting.TablePrefix,
			}

			// Get a database connection pool
			pool, err := p.GetDBPool(context.Background(), cfg, sess)
			if err != nil {
				LOGGER.Error("ERROR WHILE INITIALIZING DB POOL : ", err)
				serviceConfig.PrintSettings()
				awsClient.SendAlertMessage("FAILED", fmt.Sprintf("ERROR WHILE INITIALIZING DB POOL - %s", err))
				LOGGER.Info("Starting invalid upload file Wait")
				invalidGoroutinesWaitGroup := sync.WaitGroup{}
				invalidGoroutinesWaitGroup.Add(1)
				invalidBaseDir := utils.GetInvalidBaseDir()
				fileNameWithoutExt, _ := utils.GetFileName()
				uploadInvalidFileToS3IfExist(&invalidGoroutinesWaitGroup,
					filepath.Join(invalidBaseDir, fileNameWithoutExt+"_invalid.csv"), utils.GetRelativeInvalidBaseDir())
				LOGGER.Info("*******INVALID FILE UPLOAD CALL DONE*******")
				LOGGER.Info("Ended invalidGoroutinesWaitGroup Wait")

				//UPLOADING EXISTENCE ROW COUNT

				LOGGER.Info("Starting existence count upload file")
				existenceGoroutinesWaitGroup := sync.WaitGroup{}
				existenceGoroutinesWaitGroup.Add(1)

				uploadInvalidFileToS3IfExist(&existenceGoroutinesWaitGroup,
					existRowCountFilePath, utils.GetRelativeResultBaseDir())
				LOGGER.Info("*******EXISTENCE ROW COUNT FILE UPLOAD CALL DONE*******")
				LOGGER.Info("Ended exist count upload file")

				break
			}

			// Close the pool when you're done with it
			defer pool.Close()

			job, err := job.GetJob()
			if err != nil {
				serviceConfig.PrintSettings()
				LOGGER.Error(err)
				break
			}
			job.ExecuteStrategy(serviceConfig.ApplicationSetting.ObjectKey, "initial_offer")

			fileNameWithoutExt, _ = utils.GetFileName()
			fileNameWithoutExt += "_valid"
			failureFilePathFormat, successFilePathFormat, failurePatternString, successPatternString := getSuccessAndFailurePathFormat()
			failurePattern = regexp.MustCompile(fmt.Sprintf(failurePatternString, fileNameWithoutExt, serviceConfig.ApplicationSetting.JobType))
			successPattern = regexp.MustCompile(fmt.Sprintf(successPatternString, fileNameWithoutExt, serviceConfig.ApplicationSetting.JobType))
			jobRowCountFilePath, err := consolidation.Consolidate(failurePattern, successPattern, failureFilePathFormat, successFilePathFormat, fileNameWithoutExt, "job_row_count.csv", 5)
			if err != nil {
				LOGGER.Error("ERROR WHILE CONSOLIDATION : ", err)
				serviceConfig.PrintSettings()
				awsClient.SendAlertMessage("FAILED", fmt.Sprintf("ERROR WHILE CONSOLIDATION - %s", err))
				LOGGER.Info("Starting invalid upload file Wait")
				invalidGoroutinesWaitGroup := sync.WaitGroup{}
				invalidGoroutinesWaitGroup.Add(1)
				invalidBaseDir := utils.GetInvalidBaseDir()
				fileNameWithoutExt, _ := utils.GetFileName()
				uploadInvalidFileToS3IfExist(&invalidGoroutinesWaitGroup,
					filepath.Join(invalidBaseDir, fileNameWithoutExt+"_invalid.csv"), utils.GetRelativeInvalidBaseDir())
				LOGGER.Info("*******INVALID FILE UPLOAD CALL DONE*******")
				LOGGER.Info("Ended invalidGoroutinesWaitGroup Wait")

				//UPLOADING EXISTENCE ROW COUNT

				LOGGER.Info("Starting existence count upload file")
				existenceGoroutinesWaitGroup := sync.WaitGroup{}
				existenceGoroutinesWaitGroup.Add(1)

				uploadInvalidFileToS3IfExist(&existenceGoroutinesWaitGroup,
					existRowCountFilePath, utils.GetRelativeResultBaseDir())
				LOGGER.Info("*******EXISTENCE ROW COUNT FILE UPLOAD CALL DONE*******")
				LOGGER.Info("Ended exist count upload file")

				break
			}

			jobVerifier := &consolidation.JobVerifyConsolidatorImpl{}
			result = jobVerifier.VerifyCount(jobRowCountFilePath)
			if result.IsValid {
				LOGGER.Info("Validation passed for Result")
				awsClient.SendAlertMessage("SUCCESS", "")
			} else {
				LOGGER.Error("Row Counts Validation failed for Result")
				awsClient.SendAlertMessage("FAILED", "")
			}

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

			LOGGER.Info("**********CLOSING POOL************")

			LOGGER.Info("Starting upload invalid upload file")
			invalidGoroutinesWaitGroup := sync.WaitGroup{}
			invalidGoroutinesWaitGroup.Add(1)
			invalidBaseDir := utils.GetInvalidBaseDir()
			fileNameWithoutExt, _ = utils.GetFileName()
			uploadInvalidFileToS3IfExist(&invalidGoroutinesWaitGroup,
				filepath.Join(invalidBaseDir, fileNameWithoutExt+"_invalid.csv"), utils.GetRelativeInvalidBaseDir())
			LOGGER.Info("*******INVALID FILE UPLOAD CALL DONE*******")
			LOGGER.Info("Ended invalid upload file")

			//UPLOADING EXISTENCE ROW COUNT

			LOGGER.Info("Starting existence count upload file")
			existenceGoroutinesWaitGroup := sync.WaitGroup{}
			existenceGoroutinesWaitGroup.Add(1)

			uploadInvalidFileToS3IfExist(&existenceGoroutinesWaitGroup,
				existRowCountFilePath, utils.GetRelativeResultBaseDir())
			LOGGER.Info("*******EXISTENCE ROW COUNT FILE UPLOAD CALL DONE*******")
			LOGGER.Info("Ended exist count upload file")

			//UPLOADING JOB ROW COUNT

			LOGGER.Info("Starting job count upload file")
			jobGoroutinesWaitGroup := sync.WaitGroup{}
			jobGoroutinesWaitGroup.Add(1)

			uploadInvalidFileToS3IfExist(&jobGoroutinesWaitGroup,
				jobRowCountFilePath, utils.GetRelativeResultBaseDir())
			LOGGER.Info("*******JOB ROW COUNT FILE UPLOAD CALL DONE*******")
			LOGGER.Info("Ended JOB upload file")

		}
	}
	return nil
}

func uploadInvalidFileToS3IfExist(invalidGoroutinesWaitGroup *sync.WaitGroup, filePath, baseDir string) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("*******UPLOADING INVALID FILE*******")

	defer invalidGoroutinesWaitGroup.Done()
	ctx := context.Background()

	// AWS config
	cfg, err := multipartUpload.NewConfig(ctx)
	if err != nil {
		LOGGER.Error(err)
	}

	// AWS S3 client
	s3 := multipartUpload.NewS3(cfg, serviceConfig.ApplicationSetting.BucketName)

	if err := awsClient.UploadDriver(ctx, s3, filePath, baseDir); err != nil {
		LOGGER.Error(err)
	}
}

func setConfigFromSqsMessage(jsonMessage string) error {
	LOGGER := customLogger.GetLogger()

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

	jobType := utils.GetJobTypeFromPath()
	serviceConfig.ApplicationSetting.JobType = jobType

	invalidObjectKey := utils.GetInvalidObjectKey()
	serviceConfig.ApplicationSetting.InvalidObjectKey = invalidObjectKey

	lpc := s3UploadEvent.LPC
	serviceConfig.ApplicationSetting.Lpc = lpc
	serviceConfig.Set("lpc", lpc)

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

func getSuccessAndFailurePathFormat() (string, string, string, string) {
	if serviceConfig.ApplicationSetting.JobType == "insert" {
		failureFilePathFormat := "%s_%s_exist_failure_insert_failure.csv"
		successFilePathFormat := "%s_%s_exist_failure_insert_success.csv"
		failurePatternString := `^%s_\d+_exist_failure_%s_failure\.csv$`
		successPatternString := `^%s_\d+_exist_failure_%s_success\.csv$`
		return failureFilePathFormat, successFilePathFormat, failurePatternString, successPatternString
	}

	failureFilePathFormat := "%s_%s_exist_success_" + serviceConfig.ApplicationSetting.JobType + "_failure.csv"
	successFilePathFormat := "%s_%s_exist_success_" + serviceConfig.ApplicationSetting.JobType + "_success.csv"
	failurePatternString := `^%s_\d+_exist_success_%s_failure\.csv$`
	successPatternString := `^%s_\d+_exist_success_%s_success\.csv$`
	return failureFilePathFormat, successFilePathFormat, failurePatternString, successPatternString
}

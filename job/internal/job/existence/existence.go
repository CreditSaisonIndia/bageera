package existence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/database"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/job"
	"github.com/CreditSaisonIndia/bageera/internal/job/existence/producer"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/splitter"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type Existence struct {
	LOGGER *zap.SugaredLogger
}

func (e *Existence) ExecuteJob(path string, tableName string) error {

	// Initialize the global CustomDBManager from the new package
	// database.InitSqlxDb()
	// Create a peer instance
	// Adjust the logger as needed
	// Set to true if you want to use IAM role authentication
	// Define your database configuration
	// Adjust the port as needed
	// Adjust the SSL mode as needed
	// Set to true if you want to use IAM role authentication
	// Get a database connection pool
	// invalidBaseDir := utils.GetInvalidBaseDir()
	// fileNameWithoutExt, _ := utils.GetFileName()
	// uploadInvalidFileToS3IfExist(&invalidGoroutinesWaitGroup,
	// 	filepath.Join(invalidBaseDir, fileNameWithoutExt+"_invalid.csv"))
	pool, err := e.GetDbPool(serviceConfig.DatabaseSetting.ReaderDbHost)
	if err != nil {
		return err
	}

	e.LOGGER.Info("*******SPLITTING*******")

	err = splitter.SplitCsv(utils.GetExistenceChunksDir())
	if err != nil {
		serviceConfig.PrintSettings()
		e.LOGGER.Error("ERROR WHILE SPLITTING CSV : ", err)
		awsClient.SendAlertMessage("FAILED", fmt.Sprintf("ERROR WHILE SPLITTING CSV - %s", err))
		return err
	}

	e.LOGGER.Info("***** STARTING EXISTENCE JOB *****")
	var wg sync.WaitGroup
	var consumerWg sync.WaitGroup
	outputChunkDir := utils.GetExistenceChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		e.LOGGER.Error("Error reading directory:", err)
		return err
	}

	for index, file := range files {
		chunkDirPath := filepath.Join(outputChunkDir, strconv.Itoa(index+1))
		// Create directory with read-write-execute permissions for owner, group, and others
		err := os.MkdirAll(chunkDirPath, 0777)
		if err != nil {
			e.LOGGER.Error("Error creating chunk directory:", err)
			return err
		}
		fileUtilityWrapper.Move(filepath.Join(outputChunkDir, file.Name()), filepath.Join(chunkDirPath, file.Name()))
		wg.Add(1)
		go producer.Worker(chunkDirPath, file.Name(), &wg, &consumerWg)
	}

	// Wait for all workers to finish
	wg.Wait()
	consumerWg.Wait()

	// Close the pool when you're done with it
	pool.Close()

	e.LOGGER.Info("******CLOSING DB POOL******")
	time.Sleep(5 * time.Second)

	return nil
}

/*
Does start the insertion job for metadata table
*/
func (e *Existence) DoInsert() error {
	pool, err := e.GetDbPool(serviceConfig.DatabaseSetting.MasterDbHost)
	if err != nil {
		return err
	}

	insertionJob := &insertion.Insertion{}
	insertionJob.SetFileNamePattern(`.*_\d+_exist_success\.csv`)
	jobStrategy := job.SetStrategy(insertionJob)
	jobStrategy.ExecuteStrategy(serviceConfig.ApplicationSetting.ObjectKey, "initial_offer_history")
	pool.Close()
	return nil
}

func (e *Existence) GetDbPool(host string) (*pgxpool.Pool, error) {
	e.LOGGER.Info("SETTING SESSION FOR DATABASE")
	opts := session.Options{Config: aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(serviceConfig.ApplicationSetting.Region),
		MaxRetries:                    aws.Int(3),
	}}
	sess := session.Must(session.NewSessionWithOptions(opts))

	e.LOGGER.Info("DONE SETTING SESSION FOR DATABASE")

	p := &database.Peer{
		Name:        "peer",
		Logger:      customLogger.GetLogger(),
		IAMRoleAuth: true,
	}

	cfg := database.DBConfig{
		Host:        host,
		Port:        serviceConfig.DatabaseSetting.Port,
		User:        serviceConfig.DatabaseSetting.User,
		Password:    serviceConfig.DatabaseSetting.Password,
		SSLMode:     serviceConfig.DatabaseSetting.SslMode,
		Name:        serviceConfig.DatabaseSetting.Name,
		Region:      os.Getenv("region"),
		IAMRoleAuth: true,
		Env:         os.Getenv("environment"),
		SearchPath:  serviceConfig.DatabaseSetting.TablePrefix,
	}

	pool, err := p.GetDBPool(context.Background(), cfg, sess)
	if err != nil {
		e.LOGGER.Error("ERROR WHILE INITIALIZING DB POOL : ", err)
		serviceConfig.PrintSettings()
		awsClient.SendAlertMessage("FAILED", fmt.Sprintf("ERROR WHILE INITIALIZING DB POOL - %s", err))
		e.LOGGER.Info("Starting invalid upload file Wait")
		invalidGoroutinesWaitGroup := sync.WaitGroup{}
		invalidGoroutinesWaitGroup.Add(1)

		e.LOGGER.Info("*******INVALID FILE UPLOAD CALL DONE*******")
		e.LOGGER.Info("Ended invalidGoroutinesWaitGroup Wait")
		return nil, fmt.Errorf("ERROR WHILE INITIALIZING DB POOL : %w", err)
	}
	return pool, nil
}

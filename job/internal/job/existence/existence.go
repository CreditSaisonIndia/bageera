package existence

import (
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

	peer := &database.Peer{
		Name:        "peer",
		Logger:      customLogger.GetLogger(), // Adjust the logger as needed
		IAMRoleAuth: true,                     // Set to true if you want to use IAM role authentication
	}
	pool, err := peer.GetDbPool(serviceConfig.DatabaseSetting.ReaderDbHost)
	if err != nil {
		return err
	}

	e.LOGGER.Info("*******SPLITTING*******")

	err = splitter.SplitCsv(10000000)
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

	if serviceConfig.ApplicationSetting.JobType == "delete" {
		err = e.DoInsert()
	}

	return nil
}

/*
Does start the insertion job for metadata table
*/
func (e *Existence) DoInsert() error {
	peer := &database.Peer{
		Name:        "peer",
		Logger:      customLogger.GetLogger(), // Adjust the logger as needed
		IAMRoleAuth: true,                     // Set to true if you want to use IAM role authentication
	}
	pool, err := peer.GetDbPool(serviceConfig.DatabaseSetting.MasterDbHost)
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

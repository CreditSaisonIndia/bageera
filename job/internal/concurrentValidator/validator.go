package concurrentValidator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/concurrentValidator/consumer"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Validator struct {
	LOGGER *zap.SugaredLogger
}

// func Validate(filePath string) (bool, error) {
// 	LOGGER := customLogger.GetLogger()
// 	anyValidRow := false
// 	anyInvalidRow := false

// 	startTime := time.Now()
// 	LOGGER.Info("**********Starting validation phase**********")

// 	fileNameWithoutExt, _ := utils.GetFileName()
// 	validOutputFileName := filepath.Join(utils.GetMetadataBaseDir(), fileNameWithoutExt+"_valid.csv")
// 	invalidOutputFileDir := utils.GetInvalidBaseDir()
// 	invalidOutputFileName := filepath.Join(invalidOutputFileDir, fileNameWithoutExt+"_invalid.csv")

// 	LOGGER.Debug("ValidOutputFilePath:", validOutputFileName)
// 	LOGGER.Debug("InvalidOutputFilePath:", invalidOutputFileName)

// 	// Open input file
// 	inputFile, err := os.Open(filePath)
// 	if err != nil {
// 		LOGGER.Error("Error while opening inputFile:", err)
// 		return false, err
// 	}
// 	defer inputFile.Close()

// 	// Open valid output file
// 	LOGGER.Debug("Creating validOutputFile:", validOutputFileName)
// 	validOutputFile, err := os.Create(validOutputFileName)
// 	if err != nil {
// 		LOGGER.Error("Error while Creating validOutputFile:", err)
// 		return false, err
// 	}

// 	// Open invalid output file
// 	LOGGER.Debug("Creating invalidOutputFileName:", invalidOutputFileName)
// 	err = fileUtilityWrapper.CreateDirIfDoesNotExist(invalidOutputFileDir)
// 	if err != nil {
// 		LOGGER.Debug("Error while creating invalidOutputFileDir:", err)
// 		return false, err
// 	}
// 	invalidOutputFile, err := os.Create(invalidOutputFileName)
// 	if err != nil {
// 		LOGGER.Error("Error while Creating invalidOutputFile:", err)
// 		return false, err
// 	}

// 	// Create CSV reader for input file
// 	reader := csv.NewReader(inputFile)

// 	// Create CSV writer for output file
// 	validWriter := csv.NewWriter(validOutputFile)

// 	invalidWriter := csv.NewWriter(invalidOutputFile)

// 	// Read the header from the input file and write it to the invalid output file with the new "remarks" column
// 	header, err := reader.Read()
// 	if err != nil {
// 		LOGGER.Error("Unable to readb the CSV File:", err)
// 		return false, err
// 	}

// 	validatorFactory := GetOfferValidatorFactory(utils.GetLPC())

// 	// err = validatorFactory.validateHeader(header)
// 	// if err != nil {
// 	// 	LOGGER.Error("invalid headers:", err)
// 	// 	return false, err
// 	// }

// 	err = validWriter.Write(header)
// 	if err != nil {
// 		return false, err
// 	}

// 	header = append(header, "remarks")
// 	err = invalidWriter.Write(header)
// 	if err != nil {
// 		return false, err
// 	}

// 	LOGGER.Debug("**********Headers are written**********")

// 	for {
// 		row, err := reader.Read()
// 		if err != nil {
// 			LOGGER.Error(err)
// 			if err == io.EOF {
// 				LOGGER.Info("Successfully read the file. EOF Reached")
// 			} else {
// 				awsClient.SendAlertMessage("FAILED", "Error reading the input File")
// 			}
// 			break
// 		}
// 		isValid, remarks := validatorFactory.validateRow(row)
// 		if isValid {
// 			writeToFile(validWriter, row)
// 			if !anyValidRow {
// 				anyValidRow = !anyValidRow
// 			}
// 		} else {
// 			row_remarks := append(row, remarks)
// 			writeToFile(invalidWriter, row_remarks)
// 			if !anyInvalidRow {
// 				anyInvalidRow = !anyInvalidRow
// 			}
// 		}
// 	}
// 	invalidWriter.Flush()
// 	invalidOutputFile.Close()
// 	validWriter.Flush()
// 	validOutputFile.Close()

// 	LOGGER.Info("Validation completed. Results written to", invalidOutputFile, validOutputFile)

// 	endTime := time.Now()
// 	elapsedTime := endTime.Sub(startTime)
// 	elapsedMinutes := elapsedTime.Minutes()
// 	LOGGER.Info(fmt.Sprintf("Time taken: %.2f minutes\n", elapsedMinutes))

// 	if !anyValidRow {
// 		LOGGER.Error("No valid rows present after validation")
// 		awsClient.SendAlertMessage("FAILED", "No valid rows present after validation")
// 	} else if anyInvalidRow {
// 		LOGGER.Info("Invalid rows found after validation")
// 		awsClient.SendAlertMessage("PARTIAL FAILURE", "Invalid rows found after validation")
// 	}

// 	return !anyValidRow, nil
// }

func (v *Validator) DoValidate() error {

	startTime := time.Now()
	v.LOGGER.Info("**********Starting validation phase**********")

	outputChunkDir := utils.GetChunksDir()
	files, err := os.ReadDir(outputChunkDir)
	if err != nil {
		v.LOGGER.Error("Error reading directory:", err)
		return err
	}
	ctx := context.Background()
	consumerConfig := consumer.ConsumerConfig{

		Workers: 15,
		Retry:   3,
	}
	erg, erc := errgroup.WithContext(ctx)

	consumer := &consumer.Consumer{

		Config:           consumerConfig,
		GoroutineGroup:   erg,
		GoroutineContext: erc,
		Workers:          make(chan struct{}, consumerConfig.Workers),
		Mux:              &sync.Mutex{},
		Id:               "id",
	}
	waitGroup := sync.WaitGroup{}

	for index, file := range files {
		chunkDirPath := filepath.Join(outputChunkDir, strconv.Itoa(index+1))
		// Create directory with read-write-execute permissions for owner, group, and others
		err := os.MkdirAll(chunkDirPath, 0777)
		if err != nil {
			v.LOGGER.Error("Error creating chunk directory:", err)
			return err
		}
		fileUtilityWrapper.Move(filepath.Join(outputChunkDir, file.Name()), filepath.Join(chunkDirPath, file.Name()))
		waitGroup.Add(1)
		consumer.DoWork(chunkDirPath, file.Name(), &waitGroup)
	}
	waitGroup.Wait()
	// Wait for all workers to finish

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	elapsedMinutes := elapsedTime.Minutes()
	v.LOGGER.Info(fmt.Sprintf("Time taken: %.2f minutes\n", elapsedMinutes))

	// if !anyValidRow {
	// 	LOGGER.Error("No valid rows present after validation")
	// 	awsClient.SendAlertMessage("FAILED", "No valid rows present after validation")
	// } else if anyInvalidRow {
	// 	LOGGER.Info("Invalid rows found after validation")
	// 	awsClient.SendAlertMessage("PARTIAL FAILURE", "Invalid rows found after validation")
	// }

	return nil
}

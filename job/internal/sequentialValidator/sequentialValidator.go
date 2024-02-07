package sequentialValidator

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

var LPC = serviceConfig.ApplicationSetting.Lpc

func Validate(filePath string) (bool, error) {
	LOGGER := customLogger.GetLogger()
	anyValidRow := false
	anyInvalidRow := false

	startTime := time.Now()
	LOGGER.Info("**********Starting validation phase**********")

	fileNameWithoutExt, _ := utils.GetFileName()
	validOutputFileName := filepath.Join(utils.GetMetadataBaseDir(), fileNameWithoutExt+"_valid.csv")
	invalidOutputFileDir := utils.GetInvalidBaseDir()
	invalidOutputFileName := filepath.Join(invalidOutputFileDir, fileNameWithoutExt+"_invalid.csv")

	LOGGER.Debug("ValidOutputFilePath:", validOutputFileName)
	LOGGER.Debug("InvalidOutputFilePath:", invalidOutputFileName)

	// Open input file
	inputFile, err := os.Open(filePath)
	if err != nil {
		LOGGER.Error("Error while opening inputFile:", err)
		return false, err
	}
	defer inputFile.Close()

	// Open valid output file
	LOGGER.Debug("Creating validOutputFile:", validOutputFileName)
	validOutputFile, err := os.Create(validOutputFileName)
	if err != nil {
		LOGGER.Error("Error while Creating validOutputFile:", err)
		return false, err
	}

	// Open invalid output file
	LOGGER.Debug("Creating invalidOutputFileName:", invalidOutputFileName)
	err = fileUtilityWrapper.CreateDirIfDoesNotExist(invalidOutputFileDir)
	if err != nil {
		LOGGER.Debug("Error while creating invalidOutputFileDir:", err)
		return false, err
	}
	invalidOutputFile, err := os.Create(invalidOutputFileName)
	if err != nil {
		LOGGER.Error("Error while Creating invalidOutputFile:", err)
		return false, err
	}

	// Create CSV reader for input file
	reader := csv.NewReader(inputFile)

	// Create CSV writer for output file
	validWriter := csv.NewWriter(validOutputFile)

	invalidWriter := csv.NewWriter(invalidOutputFile)

	// Read the header from the input file and write it to the invalid output file with the new "remarks" column
	header, err := reader.Read()
	if err != nil {
		LOGGER.Error("Unable to readb the CSV File:", err)
		return false, err
	}

	validatorFactory := GetOfferValidatorFactory(serviceConfig.ApplicationSetting.Lpc)

	// err = validatorFactory.validateHeader(header)
	// if err != nil {
	// 	LOGGER.Error("invalid headers:", err)
	// 	return false, err
	// }

	err = validWriter.Write(header)
	if err != nil {
		return false, err
	}

	header = append(header, "remarks")
	err = invalidWriter.Write(header)
	if err != nil {
		return false, err
	}

	LOGGER.Debug("**********Headers are written**********")

	for {
		row, err := reader.Read()
		if err != nil {
			LOGGER.Error(err)
			if err == io.EOF {
				LOGGER.Info("Successfully read the file. EOF Reached")
			} else {
				awsClient.SendAlertMessage("FAILED", "Error reading the input File")
			}
			break
		}
		_, remarks := validatorFactory.validateRow(row)
		if true {
			writeToFile(validWriter, row)
			if !anyValidRow {
				anyValidRow = !anyValidRow
			}
		} else {
			row_remarks := append(row, remarks)
			writeToFile(invalidWriter, row_remarks)
			if !anyInvalidRow {
				anyInvalidRow = !anyInvalidRow
			}
		}
	}
	invalidWriter.Flush()
	invalidOutputFile.Close()
	validWriter.Flush()
	validOutputFile.Close()

	LOGGER.Info("Validation completed. Results written to", invalidOutputFile, validOutputFile)

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	elapsedMinutes := elapsedTime.Minutes()
	LOGGER.Info(fmt.Sprintf("Time taken: %.2f minutes\n", elapsedMinutes))

	if !anyValidRow {
		awsClient.SendAlertMessage("FAILED", "No valid rows found after validation")
	} else if anyInvalidRow {
		awsClient.SendAlertMessage("PARTIAL FAILURE", "Invalid rows found")
	}

	return !anyValidRow, nil
}

func writeToFile(writer *csv.Writer, row []string) {
	LOGGER := customLogger.GetLogger()
	err := writer.Write(row)
	if err != nil {
		LOGGER.Error(err)
		return
	}
}

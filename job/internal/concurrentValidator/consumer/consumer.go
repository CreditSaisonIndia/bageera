package consumer

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient"
	"github.com/CreditSaisonIndia/bageera/internal/concurrentValidator/validator"
	validatorimpl "github.com/CreditSaisonIndia/bageera/internal/concurrentValidator/validatorImpl"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"golang.org/x/sync/errgroup"
)

type ConsumerConfig struct {
	Workers int

	Retry int
}

type Consumer struct {
	Config           ConsumerConfig
	GoroutineGroup   *errgroup.Group
	GoroutineContext context.Context
	Workers          chan struct{}
	Mux              *sync.Mutex
	parts            map[int]*string
	Id               string
}

func (c *Consumer) DoWork(chunkDirPath, fileName string, waitGroup *sync.WaitGroup) {
	LOGGER := customLogger.GetLogger()

	c.Workers <- struct{}{}

	c.GoroutineGroup.Go(func() error {
		err := c.Worker(chunkDirPath, fileName, waitGroup)
		if err != nil {
			LOGGER.Error("Error while upload")
		}

		<-c.Workers

		return err
	})

}

func (c *Consumer) Worker(filePath, fileName string, waitGroup *sync.WaitGroup) error {

	LOGGER := customLogger.GetLogger()
	defer waitGroup.Done()
	var (
		try int
		err error
	)

	for try <= c.Config.Retry {
		select {
		case <-c.GoroutineContext.Done():
			return c.GoroutineContext.Err()
		default:
		}

		err = validateFile(fileName, filePath)
		if err == nil {
			LOGGER.Info("fileName Done --- > ", fileName)
			c.Mux.Lock()
			//c.parts[0] = res.ETag
			c.Mux.Unlock()

			return nil
		}
		LOGGER.Infof("******RETRYING VLDAITION FOR  : %s --- > ", fileName)
		try++
	}

	return fmt.Errorf("Error upload part: %w", err)

}

func validateFile(fileName string, filePath string) error {
	LOGGER := customLogger.GetLogger()

	LOGGER.Infof("***********STARTING WORKER FOR FILE : %s *************", fileName)
	anyValidRow := false
	anyInvalidRow := false

	fileNameWithoutExt, _ := utils.GetFileNameFromPath(fileName)
	validOutputFileName := filepath.Join(filePath, fileNameWithoutExt+"_valid.csv")

	invalidOutputFileName := filepath.Join(filePath, fileNameWithoutExt+"_invalid.csv")

	LOGGER.Debug("ValidOutputFilePath:", validOutputFileName)
	LOGGER.Debug("InvalidOutputFilePath:", invalidOutputFileName)

	inputFile, err := os.Open(filepath.Join(filePath, fileName))
	if err != nil {
		LOGGER.Error("Error while opening inputFile:", err)
		return err

	}
	defer inputFile.Close()

	LOGGER.Debug("Creating validOutputFile:", validOutputFileName)
	validOutputFile, err := os.Create(validOutputFileName)
	if err != nil {
		LOGGER.Error("Error while Creating validOutputFile:", err)
		return err
	}

	invalidOutputFile, err := os.Create(invalidOutputFileName)
	if err != nil {
		LOGGER.Error("Error while Creating invalidOutputFile:", err)
		return err
	}

	reader := csv.NewReader(inputFile)

	validWriter := csv.NewWriter(validOutputFile)

	invalidWriter := csv.NewWriter(invalidOutputFile)

	header, err := reader.Read()
	if err != nil {
		LOGGER.Error("Unable to readb the CSV File:", err)
		return err
	}

	validatorFactory := GetOfferValidatorFactory(serviceConfig.ApplicationSetting.Lpc)

	err = validatorFactory.ValidateHeader(header)
	if err != nil {
		LOGGER.Error("invalid headers:", err)

	}

	err = validWriter.Write(header)
	if err != nil {
		return err
	}

	header = append(header, "remarks")
	err = invalidWriter.Write(header)
	if err != nil {
		return err
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
		isValid, remarks := validatorFactory.ValidateRow(row)
		if isValid {
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

	LOGGER.Infof("Validation completed :  %s   |  Results written : %s", invalidOutputFile.Name(), validOutputFile.Name())
	return nil
}

func writeToFile(writer *csv.Writer, row []string) {
	LOGGER := customLogger.GetLogger()
	err := writer.Write(row)
	if err != nil {
		LOGGER.Error(err)
		return
	}
}

func GetOfferValidatorFactory(lpc string) validator.OfferValidatorInterface {
	switch lpc {
	case "PSB", "ONL", "SPM":
		return &validatorimpl.JsonOfferValidatorFactory{}
	default:
		return &validatorimpl.ColumnOfferValidatorFactory{}
	}
}

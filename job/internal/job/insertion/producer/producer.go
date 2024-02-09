package producer

import (
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/csvUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion/consumer"
	"github.com/CreditSaisonIndia/bageera/internal/reader"
	readerIml "github.com/CreditSaisonIndia/bageera/internal/reader/readerImpl"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

var maxProducerGoroutines = 15
var ProducerConcurrencyCh = make(chan struct{}, maxProducerGoroutines)

func Worker(outputDir string, fileName string, wg *sync.WaitGroup, consumerWg *sync.WaitGroup) {
	LOGGER := customLogger.GetLogger()
	defer wg.Done()
	ProducerConcurrencyCh <- struct{}{}

	filePath := filepath.Join(outputDir, fileName)
	LOGGER.Info("Starting producer for : ", filePath)
	reader, err := fileUtilityWrapper.CreateReader(filePath)
	if err != nil {
		LOGGER.Error("Error creating reader:", err)
		return
	}

	defer func() {
		// Close the file only when processing is complete
		if closer, ok := reader.(io.Closer); ok {
			err := closer.Close()
			if err != nil {
				LOGGER.Error("Error closing file:", err)
			}
		}
	}()

	// Create a CSV reader
	csvReader := csvUtilityWrapper.NewCSVReader(reader)
	// Read the CSV file and send chunks to the channel
	header, err := csvReader.Read()
	if err != nil {
		LOGGER.Error("Headers", header, err)
	}

	offerReader := getReaderType(csvReader)
	offerReader.SetHeader(header)

	offersPointer, err := offerReader.ReaderStrategy(csvReader)
	offerLength := len(*offersPointer)
	if err == io.EOF {
		LOGGER.Error("Reached End of the file with overall size of : ", offerLength)
	} else if err != nil {
		LOGGER.Error("Error while reading fileName: : ", fileName, err)
		LOGGER.Error("Producer finished : ", filePath)
		<-ProducerConcurrencyCh
		return
	}

	s := fmt.Sprintf(filePath+" |  Chunk size %v", offerLength)
	LOGGER.Info(s)
	consumerWg.Add(1)

	consumer.Worker(outputDir, fileName, offersPointer, consumerWg, header)

	LOGGER.Info("Producer finished : ", filePath)
	<-ProducerConcurrencyCh

}

func getReaderType(csvReader *csv.Reader) *reader.Reader {

	switch serviceConfig.ApplicationSetting.Lpc {
	case "PSB", "ONL", "SPM":
		psbOfferCsvReader := &readerIml.PsbOfferCsvReader{}

		return reader.SetReader(psbOfferCsvReader)

	case "GRO", "ANG":
		groCsvOfferReader := &readerIml.GroCsvOfferReader{}

		return reader.SetReader(groCsvOfferReader)

	default:
		singleOfferReader := &readerIml.SingleOfferReader{}

		return reader.SetReader(singleOfferReader)
	}
}

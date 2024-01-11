package producer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/CreditSaisonIndia/bageera/internal/csvUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/job/insertion/consumer"
	"github.com/CreditSaisonIndia/bageera/internal/model"
)

var maxProducerGoroutines = 50
var ProducerConcurrencyCh = make(chan struct{}, maxProducerGoroutines)

func Worker(outputDir string, fileName string, wg *sync.WaitGroup, consumerWg *sync.WaitGroup) {
	LOGGER := customLogger.GetLogger()
	defer wg.Done()
	ProducerConcurrencyCh <- struct{}{}

	filePath := filepath.Join(outputDir, fileName)
	LOGGER.Info("Starting producer for : ", filePath)
	reader, err := fileUtilityWrapper.CreateReader(filePath)
	if err != nil {
		LOGGER.Info("Error creating reader:", err)
		return
	}

	defer func() {
		// Close the file only when processing is complete
		if closer, ok := reader.(io.Closer); ok {
			err := closer.Close()
			if err != nil {
				LOGGER.Info("Error closing file:", err)
			}
		}
	}()

	// Create a CSV reader
	csvReader := csvUtilityWrapper.NewCSVReader(reader)
	// Read the CSV file and send chunks to the channel
	header, err := csvReader.Read()
	if err != nil {
		LOGGER.Info("Headers", header)
	}

	offers, err := ReadOffers(csvReader)
	if err == io.EOF {
		LOGGER.Info("Reached End of the file with overall size of : ", len(offers))
	} else if err != nil {
		LOGGER.Info("Error while reading fileName: : ", fileName)
		LOGGER.Info("Producer finished : ", filePath)
		<-ProducerConcurrencyCh
		return
	}

	s := fmt.Sprintf(filePath+" |  Chunk size %v", len(offers))
	LOGGER.Info(s)
	LOGGER.Info("Chunk 1st parnterLoanId   " + offers[0].PartnerLoanID)
	LOGGER.Info("Chunk Last parnterLoanId   " + offers[len(offers)-1].PartnerLoanID)
	consumerWg.Add(1)
	consumer.Worker(outputDir, fileName, offers, consumerWg)

	LOGGER.Info("Producer finished : ", filePath)
	<-ProducerConcurrencyCh
	// Process the data in the producer if needed
	// (you can replace this with your own logic)
}

func ReadOffers(r *csv.Reader) ([]model.Offer, error) {
	LOGGER := customLogger.GetLogger()
	var offers []model.Offer

	for {
		record, err := r.Read()

		//for index, value := range record {
		//      fmt.Printf("Index: %d, Value: %s\n", index, value)
		//}
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		//// Sanitize the "offer_details" field by escaping double quotes
		//sanitizedJSON := strings.Replace(record[1], `"`, `\"`, -1)
		////
		//LOGGER.Info(sanitizedJSON)
		// Parse the sanitized "offer_details" field as JSON
		var offerDetails []model.OfferDetail
		if err := json.Unmarshal([]byte(record[1]), &offerDetails); err != nil {
			LOGGER.Info(err)
			return nil, err
		}
		offer := model.Offer{
			PartnerLoanID: record[0],
			OfferDetails:  offerDetails,
		}

		offers = append(offers, offer)
	}

	return offers, nil
}

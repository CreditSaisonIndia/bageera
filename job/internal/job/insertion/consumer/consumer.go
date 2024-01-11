package consumer

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/csvUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/database"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/jinzhu/gorm/dialects/postgres"
)

var maxConsumerGoroutines = 50
var ConsumerConcurrencyCh = make(chan struct{}, maxConsumerGoroutines)

func Worker(filePath, fileName string, offers []model.Offer, consumerWaitGroup *sync.WaitGroup) {
	LOGGER := customLogger.GetLogger()
	defer consumerWaitGroup.Done()
	ConsumerConcurrencyCh <- struct{}{}

	gormDb := database.GetDb()
	chunkFileNameWithoutExtension := filepath.Base(fileName[:len(fileName)-len(filepath.Ext(fileName))])
	LOGGER.Info("chunkFileNameWithoutExtension : ", chunkFileNameWithoutExtension)
	successFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_success.csv")
	successFile, err := os.Create(successFilePath)
	if err != nil {
		LOGGER.Fatal("Error creating success.csv:", err)
	}
	defer successFile.Close()
	failureFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_failure.csv")
	failureFile, err := os.Create(failureFilePath)
	if err != nil {
		LOGGER.Fatal("Error creating failure.csv:", err)
	}
	defer failureFile.Close()

	// Create CSV writers
	successWriter := csv.NewWriter(successFile)
	defer successWriter.Flush()

	header := []string{"partner_loan_id", "offer_details"}
	if err := successWriter.Write(header); err != nil {
		LOGGER.Info("Error writing CSV header:", err)
		return
	}

	failureWriter := csv.NewWriter(failureFile)
	defer failureWriter.Flush()

	if err := failureWriter.Write(header); err != nil {
		LOGGER.Info("Error writing CSV header:", err)
		return
	}

	LOGGER.Info("Worker : " + fileName + " started with offer size  " + strconv.Itoa(len(offers)))
	chunkSize := 1000
	chunkNUmber := 0
	// Convert offers to DbInitialOffer and add to a slice
	for i := 0; i < len(offers); i += chunkSize {
		chunkNUmber++
		end := i + chunkSize
		if end > len(offers) {
			end = len(offers)
		}

		chunk := offers[i:end]

		var proddbOffers []model.InitialOffer
		for _, offer := range chunk {
			proddbOffer, err := convertToProddbOffer(offer)
			if err != nil {
				LOGGER.Info("Error converting offer:", err)
				csvUtilityWrapper.WriteOfferToCSV(failureWriter, offer)
				continue
			}
			proddbOffers = append(proddbOffers, proddbOffer)
		}
		// Use gormDb.Create with variadic arguments to perform a batch insert
		if err := gormDb.Create(proddbOffers).Error; err != nil {
			LOGGER.Info("Error while creating statements:", err)
			for _, offer := range offers {
				csvUtilityWrapper.WriteOfferToCSV(failureWriter, offer)
			}
			return
		}
		LOGGER.Info("------------INSERTED--------------")
		LOGGER.Info("fileName : ", fileName, "CHUNK NUMBER : ", chunkNUmber)

		for _, offer := range offers {
			csvUtilityWrapper.WriteOfferToCSV(successWriter, offer)
		}

	}

	LOGGER.Info("Worker finished :", fileName, "------> Inserted ", chunkNUmber, " times")
	<-ConsumerConcurrencyCh
}

func convertToProddbOffer(offer model.Offer) (model.InitialOffer, error) {
	LOGGER := customLogger.GetLogger()

	var proddbInitialOffer model.InitialOffer
	rawSection, err := getSection(offer.OfferDetails)
	if err != nil {
		LOGGER.Info("Error While forming marshalling section with following error : ", err)
		return proddbInitialOffer, err
	}
	proddbInitialOffer = model.InitialOffer{
		// Set other fields based on your requirements
		IsActive:      true, // For example
		IsDeleted:     false,
		UpdatedAt:     time.Now(),
		PartnerLoanID: offer.PartnerLoanID,
		Status:        sql.NullInt64{Int64: 30},
		OfferSections: postgres.Jsonb{rawSection},

		// ApplicableSegments: firstOffer.ApplicableSegments,
		// Set other fields as needed
	}

	return proddbInitialOffer, nil
}

func getSection(offers []model.OfferDetail) ([]byte, error) {
	LOGGER := customLogger.GetLogger()
	var offerSectionArray []model.OfferSection

	for _, resp := range offers {
		var internalOfferArray = resp.Offers

		for _, internalOffer := range internalOfferArray {
			section := model.OfferSection{
				ID:                 internalOffer.OfferID,
				Interest:           internalOffer.ROI,
				Tenure:             internalOffer.PreferredTenure,
				ValidTill:          resp.ExpiryDateOfOffer,
				PreApprovalDate:    resp.DateOfOffer,
				PreferredTenure:    internalOffer.PreferredTenure,
				MaxTenure:          internalOffer.MaxTenure,
				MinTenure:          internalOffer.MinTenure,
				ExpiryDateOfOffer:  resp.ExpiryDateOfOffer,
				CreditLimit:        internalOffer.CreditLimit,
				RateOfInterest:     internalOffer.ROI,
				PF:                 internalOffer.PF,
				DedupeString:       int(resp.DedupeString),
				ApplicableSegments: internalOffer.ApplicableSegments,
			}
			offerSectionArray = append(offerSectionArray, section)
		}
	}

	rawMessage, err := json.Marshal(offerSectionArray)
	if err != nil {
		LOGGER.Info("Error while marshalling to json b:", err)
		return nil, err
	}

	// Print the json.RawMessage
	return rawMessage, nil
}

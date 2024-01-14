package consumer

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/csvUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/database"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
)

var maxConsumerGoroutines = 30
var ConsumerConcurrencyCh = make(chan struct{}, maxConsumerGoroutines)

func Worker(filePath, fileName string, offers []model.Offer, consumerWaitGroup *sync.WaitGroup) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("Worker : " + fileName + " started with offer size  " + strconv.Itoa(len(offers)))
	defer consumerWaitGroup.Done()
	ConsumerConcurrencyCh <- struct{}{}

	// customDB := dbmanager.GlobalDBManager.GetDB()
	// defer dbmanager.GlobalDBManager.ReleaseDB(customDB)

	// db := customDB.DB
	// if db == nil {
	// 	// Handle the case where DB is nil
	// 	return
	// }

	// if err != nil {
	// 	LOGGER.Error("Error while database.InitSqlxDb:", err)
	// 	<-ConsumerConcurrencyCh
	// 	return
	// }

	chunkFileNameWithoutExtension := filepath.Base(fileName[:len(fileName)-len(filepath.Ext(fileName))])

	successFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_success.csv")

	successFile, err := os.Create(successFilePath)
	if err != nil {
		LOGGER.Error("Error creating success.csv:", err)
		<-ConsumerConcurrencyCh
		return
	}
	defer successFile.Close()
	failureFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_failure.csv")
	failureFile, err := os.Create(failureFilePath)
	if err != nil {
		LOGGER.Error("Error creating failure.csv:", err)
		<-ConsumerConcurrencyCh
		return
	}
	defer failureFile.Close()

	// Create CSV writers
	successWriter := csv.NewWriter(successFile)
	defer successWriter.Flush()

	header := []string{"partner_loan_id", "offer_details"}
	if err := successWriter.Write(header); err != nil {
		LOGGER.Error("Error writing CSV header:", err)
		<-ConsumerConcurrencyCh
		return
	}

	failureWriter := csv.NewWriter(failureFile)
	defer failureWriter.Flush()

	if err := failureWriter.Write(header); err != nil {
		LOGGER.Info("Error writing CSV header:", err)
		<-ConsumerConcurrencyCh
		return
	}
	col := []string{"created_at", "is_active", "is_deleted", "updated_at", "app_form_id", "partner_loan_id",
		"status", "offer_sections", "description", "remarks",
		"attempt"}

	LOGGER.Info("Worker : " + fileName + " started with offer size  " + strconv.Itoa(len(offers)))
	chunkSize := 2000
	chunkNumber := 0

	// ... (existing code)

	for i := 0; i < len(offers); i += chunkSize {
		chunkNumber++
		var (
			placeholders []string
			vals         []interface{}
		)

		chunkEnd := i + chunkSize
		if chunkEnd > len(offers) {
			chunkEnd = len(offers)
		}

		chunk := offers[i:chunkEnd]

		//var proddbOffers []model.InitialOffer
		for index, offer := range chunk {

			proddbOffer, err := convertToProddbOffer(offer)
			if err != nil {
				LOGGER.Error("Error converting offer:", err)
				csvUtilityWrapper.WriteOfferToCSV(failureWriter, offer)
				continue
			}
			//print("proddbOffer.IsActive : ", reflect.TypeOf(proddbOffer.IsActive))
			placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				index*11+1,
				index*11+2,
				index*11+3,
				index*11+4,
				index*11+5,
				index*11+6,
				index*11+7,
				index*11+8,
				index*11+9,
				index*11+10,
				index*11+11,
			))

			vals = append(vals, proddbOffer.CreatedAt, proddbOffer.IsActive, proddbOffer.IsDeleted, proddbOffer.UpdatedAt, proddbOffer.AppFormID, proddbOffer.PartnerLoanID, proddbOffer.Status, proddbOffer.OfferSections, proddbOffer.Description, proddbOffer.Remarks, proddbOffer.Attempt)
		}

		// print("placeholders : ", placeholders)
		// print("vals: ", vals)

		// Construct the SQL query
		query := fmt.Sprintf("INSERT INTO initial_offer (%s) VALUES %s", strings.Join(col, ", "), strings.Join(placeholders, ","))
		LOGGER.Info("Worker : " + fileName + "--->>>> GETTTING DB")
		// Now you can use the pool to get a database connection
		conn, err := database.GetPgxPool().Acquire(context.Background())
		if err != nil {
			LOGGER.Error("Error : Worker : "+fileName+" ------>  accquire ", err)
			for _, offer := range chunk {
				csvUtilityWrapper.WriteOfferToCSV(failureWriter, offer)
			}
			<-ConsumerConcurrencyCh
			return
		}

		tx, err := conn.Begin(context.Background())
		if err != nil {
			LOGGER.Error("Error : Worker : "+fileName+"------> conn.Begin(context.Background()) ", err)
			for _, offer := range chunk {
				csvUtilityWrapper.WriteOfferToCSV(failureWriter, offer)
			}
			<-ConsumerConcurrencyCh
			return
		}

		_, err = tx.Exec(context.Background(), query, vals...)
		if err != nil {
			LOGGER.Error("Error : Worker : "+fileName+" ------> tx.Exe ", err)
			tx.Rollback(context.Background())
			for _, offer := range chunk {
				csvUtilityWrapper.WriteOfferToCSV(failureWriter, offer)
			}
			<-ConsumerConcurrencyCh
			return
		}

		// Commit the transaction inside the loop
		err = tx.Commit(context.Background())
		if err != nil {
			LOGGER.Info("Error : ------>  committing transaction:", err)
			tx.Rollback(context.Background())
			for _, offer := range chunk {
				csvUtilityWrapper.WriteOfferToCSV(failureWriter, offer)
			}
			<-ConsumerConcurrencyCh
			return
		}

		LOGGER.Info("------------INSERTED--------------")
		LOGGER.Info("fileName : ", chunkFileNameWithoutExtension, "---->>>>>>CHUNK NUMBER : ", chunkNumber)

		for _, offer := range chunk {
			csvUtilityWrapper.WriteOfferToCSV(successWriter, offer)
		}
		LOGGER.Info("------------RELEASING CONNECTION--------------")
		conn.Release()
	}

	LOGGER.Info("Worker finished :", fileName, "------> Inserted ", chunkNumber, " times")
	<-ConsumerConcurrencyCh
}

func convertToInterfaceSlice(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case bool:
			result[i] = pq.BoolArray{val}
		default:
			result[i] = v
		}
	}
	return result
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

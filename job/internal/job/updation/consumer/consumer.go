package consumer

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/database"
	"github.com/CreditSaisonIndia/bageera/internal/job/common/parser"
	"github.com/CreditSaisonIndia/bageera/internal/job/common/parser/parserIml"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

var maxConsumerGoroutines = 15
var ConsumerConcurrencyCh = make(chan struct{}, maxConsumerGoroutines)

func Worker(filePath, fileName string, offersPointer *[]model.BaseOffer, consumerWaitGroup *sync.WaitGroup, header []string) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("Worker : " + fileName + " started with offer size  " + strconv.Itoa(len(*offersPointer)))
	defer consumerWaitGroup.Done()
	ConsumerConcurrencyCh <- struct{}{}

	// customDB := dbmanager.GlobalDBManager.GetDB()
	// defer dbmanager.GlobalDBManager.ReleaseDB(customDB)

	// db := customDB.DB
	// if db == nil {
	//  // Handle the case where DB is nil
	//  return
	// }

	// if err != nil {
	//  LOGGER.Error("Error while database.InitSqlxDb:", err)
	//  <-ConsumerConcurrencyCh
	//  return
	// }

	chunkFileNameWithoutExtension := filepath.Base(fileName[:len(fileName)-len(filepath.Ext(fileName))])

	successFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_update_success.csv")

	successFile, err := os.Create(successFilePath)
	if err != nil {
		LOGGER.Error("Error creating success.csv:", err)
		<-ConsumerConcurrencyCh
		return
	}
	defer successFile.Close()
	failureFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_update_failure.csv")
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

	parser := getParserType()

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

	chunkSize := 2000
	chunkNumber := 0

	col := []string{"updated_at", "partner_loan_id", "offer_sections", "description", "remarks", "attempt", "expiry_date"}

	// ... (existing code)
	offersLength := len(*offersPointer)
	offers := *offersPointer
	for i := 0; i < offersLength; i += chunkSize {
		chunkNumber++
		var (
			placeholders []string
			vals         []interface{}
		)
		chunkEnd := i + chunkSize
		if chunkEnd > offersLength {
			chunkEnd = offersLength
		}

		chunk := offers[i:chunkEnd]
		//var proddboffersPointer []model.InitialOffer
		for index, offer := range chunk {

			proddbOffer, err := parser.ParserStrategy(&offer)
			if err != nil {
				LOGGER.Error("Error converting offer:", err)
				parser.WriteOfferToCsv(failureWriter, &offer)
				continue
			}

			placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				index*len(col)+1,
				index*len(col)+2,
				index*len(col)+3,
				index*len(col)+4,
				index*len(col)+5,
				index*len(col)+6,
				index*len(col)+7,
			))

			vals = append(vals, proddbOffer.UpdatedAt, proddbOffer.PartnerLoanID, proddbOffer.OfferSections, proddbOffer.Description, proddbOffer.Remarks, proddbOffer.Attempt+1, proddbOffer.ExpiryDate)

		}

		print("placeholders : ", placeholders)
		print("vals: ", vals)

		// Construct the SQL query
		// Construct the SQL query with ON CONFLICT
		query := fmt.Sprintf("INSERT INTO initial_offer (%s) VALUES %s ON CONFLICT (partner_loan_id) DO UPDATE SET "+
			"updated_at = EXCLUDED.updated_at, "+
			"offer_sections = EXCLUDED.offer_sections,"+
			"description = EXCLUDED.description,"+
			"remarks = EXCLUDED.remarks,"+
			"attempt = EXCLUDED.attempt,expiry_date = EXCLUDED.expiry_date",
			strings.Join(col, ", "), strings.Join(placeholders, ","))

		LOGGER.Info("Worker : " + fileName + "--->>>> GETTTING DB")
		// Now you can use the pool to get a database connection
		conn, err := database.GetPgxPool().Acquire(context.Background())
		if err != nil {
			LOGGER.Error("Error Worker : "+fileName+" ------>  accquire ", err)
			for _, offer := range chunk {
				parser.WriteOfferToCsv(failureWriter, &offer)
			}
			<-ConsumerConcurrencyCh
			return
		}

		tx, err := conn.Begin(context.Background())
		if err != nil {
			LOGGER.Error("Error : Worker : "+fileName+"------> conn.Begin(context.Background()) ", err)
			for _, offer := range chunk {
				parser.WriteOfferToCsv(failureWriter, &offer)
			}
			<-ConsumerConcurrencyCh
			return
		}

		_, err = tx.Exec(context.Background(), query, vals...)
		if err != nil {
			LOGGER.Error("Error : Worker : "+fileName+" ------> tx.Exe ", err)
			tx.Rollback(context.Background())
			for _, offer := range chunk {
				parser.WriteOfferToCsv(failureWriter, &offer)
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
				parser.WriteOfferToCsv(failureWriter, &offer)
			}
			<-ConsumerConcurrencyCh
			return
		}

		LOGGER.Info("------------UPDATED--------------")
		LOGGER.Info("fileName : ", chunkFileNameWithoutExtension, "---->>>>>>CHUNK NUMBER : ", chunkNumber)

		for _, offer := range chunk {
			parser.WriteOfferToCsv(successWriter, &offer)
		}
		LOGGER.Info("------------RELEASING CONNECTION--------------")
		conn.Release()
		time.Sleep(5 * time.Second)
		LOGGER.Info("******** WORKER ", fileName, "  | WARMING UP TO WORK ON NEXT CHUNK ---> ", chunkNumber, " ********")
	}

	LOGGER.Info("Worker finished :", fileName, "------> UPDATED ", chunkNumber, " times")

	<-ConsumerConcurrencyCh
}

func getParserType() *parser.Parser {

	switch serviceConfig.ApplicationSetting.Lpc {
	case "PSB", "ONL":
		psbOfferParser := &parserIml.PsbOfferParser{}
		return parser.SetParser(psbOfferParser)
	case "GRO", "ANG":
		groOfferParser := &parserIml.GroOfferParser{}
		return parser.SetParser(groOfferParser)

	default:
		singleOfferParser := &parserIml.SingleOfferParser{}
		return parser.SetParser(singleOfferParser)
	}
}

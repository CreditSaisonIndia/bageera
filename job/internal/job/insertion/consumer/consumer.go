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

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/database"
	"github.com/CreditSaisonIndia/bageera/internal/job/common/parser"
	"github.com/CreditSaisonIndia/bageera/internal/job/common/parser/parserIml"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

var maxConsumerGoroutines = 15
var ConsumerConcurrencyCh = make(chan struct{}, maxConsumerGoroutines)

func Worker(filePath, fileName string, offersPointer *[]model.BaseOffer, consumerWaitGroup *sync.WaitGroup, header []string, tableName string) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("Worker : " + fileName + " started with offer size  " + strconv.Itoa(len(*offersPointer)))
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

	successFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_insert_success.csv")

	successFile, err := os.Create(successFilePath)
	if err != nil {
		LOGGER.Error("Error creating success.csv:", err)
		<-ConsumerConcurrencyCh
		return
	}
	defer successFile.Close()
	failureFilePath := filepath.Join(filePath, chunkFileNameWithoutExtension+"_insert_failure.csv")
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
	col := []string{"created_at", "is_active", "is_deleted", "updated_at", "app_form_id", "partner_loan_id",
		"status", "offer_sections", "description", "remarks", "attempt", "expiry_date"}

	chunkSize := 2000
	chunkNumber := 0

	// ... (existing code)
	offersLength := len(*offersPointer)
	offers := *offersPointer

	LOGGER.Info("Worker : " + fileName + "--->>>> GETTTING DB")
	// Now you can use the pool to get a database connection
	conn, err := database.GetPgxPool().Acquire(context.Background())
	if err != nil {
		LOGGER.Error("Error Worker : "+fileName+" ------>  accquire ", err)
		for _, offer := range offers {
			parser.WriteOfferToCsv(failureWriter, &offer)
		}
		LOGGER.Error("Error : Worker " + fileName + " ------> Erroed out. Hence written all offers to failure file")
		<-ConsumerConcurrencyCh
		return
	}
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

			//print("proddbOffer.IsActive : ", reflect.TypeOf(proddbOffer.IsActive))
			placeholders = append(placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				index*len(col)+1,
				index*len(col)+2,
				index*len(col)+3,
				index*len(col)+4,
				index*len(col)+5,
				index*len(col)+6,
				index*len(col)+7,
				index*len(col)+8,
				index*len(col)+9,
				index*len(col)+10,
				index*len(col)+11,
				index*len(col)+12,
			))

			vals = append(vals, proddbOffer.CreatedAt, proddbOffer.IsActive, proddbOffer.IsDeleted, proddbOffer.UpdatedAt, proddbOffer.AppFormID, proddbOffer.PartnerLoanID, proddbOffer.Status, proddbOffer.OfferSections, proddbOffer.Description, proddbOffer.Remarks, proddbOffer.Attempt, proddbOffer.ExpiryDate)
		}

		// print("placeholders : ", placeholders)
		// print("vals: ", vals)

		// Construct the SQL query
		// Construct the SQL query with ON CONFLICT
		// query := fmt.Sprintf("INSERT INTO initial_offer (%s) VALUES %s ON CONFLICT (partner_loan_id) DO UPDATE SET "+
		// 	"updated_at = EXCLUDED.updated_at, "+
		// 	"offer_sections = EXCLUDED.offer_sections,"+
		// 	"attempt = EXCLUDED.attempt,expiry_date = EXCLUDED.expiry_date",
		// 	strings.Join(col, ", "), strings.Join(placeholders, ","))
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", tableName, strings.Join(col, ", "), strings.Join(placeholders, ","))

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

		LOGGER.Info("------------INSERTED--------------")
		LOGGER.Info("fileName : ", chunkFileNameWithoutExtension, "---->>>>>>CHUNK NUMBER : ", chunkNumber)

		for _, offer := range chunk {
			parser.WriteOfferToCsv(successWriter, &offer)
		}

		LOGGER.Info("******** WORKER ", fileName, "  | WARMING UP TO WORK ON NEXT CHUNK ---> ", chunkNumber, " ********")
	}
	LOGGER.Info("------------RELEASING CONNECTION--------------")
	conn.Release()

	LOGGER.Info("Worker finished :", fileName, "------> Inserted ", chunkNumber, " times")

	<-ConsumerConcurrencyCh
}

func getParserType() *parser.Parser {

	switch serviceConfig.ApplicationSetting.Lpc {
	case "PSB", "ONL", "SPM":
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

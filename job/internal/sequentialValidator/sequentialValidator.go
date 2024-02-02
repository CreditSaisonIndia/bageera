package sequentialValidator

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/awsClient"
	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/fileUtilityWrapper"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/CreditSaisonIndia/bageera/internal/utils"

	"github.com/go-playground/validator/v10"
)

type OfferInfo struct {
	ApplicableSegments []int   `json:"applicable_segments" validate:"required"`
	PreferredTenure    int     `json:"preferred_tenure" validate:"required"`
	MaxTenure          int     `json:"max_tenure" validate:"required"`
	MinTenure          int     `json:"min_tenure" validate:"required"`
	PF                 float64 `json:"pf" validate:"required"`
	CreditLimit        float64 `json:"credit_limit" validate:"ValidCreditLimit"`
	LimitAmount        float64 `json:"limit_amount" validate:"ValidLimitAmount"`
	OfferID            string  `json:"offer_id" validate:"required"`
	ROI                string  `json:"roi" validate:"required"`
}

type OfferDetail struct {
	Offers            []*OfferInfo `json:"offers" validate:"required,dive,required"`
	DateOfOffer       string       `json:"date_of_offer" validate:"required,IsValidDate"`
	ExpiryDateOfOffer string       `json:"expiry_date_of_offer" validate:"required,IsValidDate"`
	DedupeString      string       `json:"dedupe_string" validate:"required"`
}

var LPC = serviceConfig.ApplicationSetting.Lpc

func IsValidDate(fl validator.FieldLevel) bool {
	dateString := fl.Field().String()
	// Define the expected date layout
	layout := "2006-01-02"

	// Parse the date string
	_, err := time.Parse(layout, dateString)

	// Check if there was an error during parsing
	return err == nil
}

func ValidLimitAmount(fl validator.FieldLevel) bool {
	LimitAmount := fl.Field().Float()
	if LPC == "PSB" {
		return LimitAmount == 0.0
	}
	return LimitAmount != 0.0
}

func ValidCreditLimit(fl validator.FieldLevel) bool {
	CreditLimit := fl.Field().Float()
	if LPC == "PSB" {
		return CreditLimit != 0.0
	}
	return CreditLimit == 0.0
}

// Validation for each row
func validateRow(row []string) (isValid bool, remarks string) {
	// Validate Number of fields in each row to be 2
	var remarks_list []string
	if len(row) != 2 {
		return false, "Invalid number of fields present in the row"
	}
	// Validate the two fields to be not empty
	if len(row[0]) == 0 {
		remarks_list = append(remarks_list, "partner_loan_id cannot be empty")
	}
	if len(row[1]) == 0 {
		remarks_list = append(remarks_list, "offer_details cannot be empty")
	}
	if len(remarks_list) != 0 {
		return false, strings.Join(remarks_list, ";")
	}

	var offerDetails []OfferDetail
	if err := json.Unmarshal([]byte(row[1]), &offerDetails); err != nil {
		remarks_list = append(remarks_list, fmt.Sprintf("Error: %s", err))
		return len(remarks_list) == 0, strings.Join(remarks_list, ";")
	}
	// Validation for multiple elements in the root list
	if len(offerDetails) != 1 {
		return false, "Invalid Number of elements at the root list"
	}

	validate := validator.New()
	validate.RegisterValidation("IsValidDate", IsValidDate)
	validate.RegisterValidation("ValidLimitAmount", ValidLimitAmount)
	validate.RegisterValidation("ValidCreditLimit", ValidCreditLimit)
	err := validate.Struct(offerDetails[0])
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			remarks_list = append(remarks_list, fmt.Sprintf("Field: %s, Error: %s", e.Field(), e.Tag()))
		}
	}

	return len(remarks_list) == 0, strings.Join(remarks_list, ";")
}

// Header validation
func validateHeader(headers []string) error {
	// validate 2 columns are present in the header
	if len(headers) != 2 {
		return fmt.Errorf("invalid headers length")
	}
	// Check if the present 2 columns are partner_loan_id and offer_details
	if headers[0] != "partner_loan_id" {
		return fmt.Errorf("invalid header column - %s", headers[0])
	}
	if headers[1] != "offer_details" {
		return fmt.Errorf("invalid header column - %s", headers[1])
	}
	return nil
}

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

	err = validateHeader(header)
	if err != nil {
		LOGGER.Error("invalid headers:", err)
		return false, err
	}

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
		isValid, remarks := validateRow(row)
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

	LOGGER.Info("Validation completed. Results written to", invalidOutputFile, validOutputFile)

	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	elapsedMinutes := elapsedTime.Minutes()
	LOGGER.Info(fmt.Sprintf("Time taken: %.2f minutes\n", elapsedMinutes))

	if !anyValidRow {
		awsClient.SendAlertMessage("FAILED", "No valid rows found")
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

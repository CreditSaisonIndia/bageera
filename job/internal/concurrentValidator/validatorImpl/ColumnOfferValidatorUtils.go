package validatorimpl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"github.com/go-playground/validator/v10"
)

var headerLength = 9

type ColumnOfferValidatorFactory struct{}

type OfferColumns struct {
	PartnerLoanId     string  ` validate:"required"`
	OfferID           string  `validate:"required"`
	CreditLimit       float64 ` validate:"required"`
	MinTenure         int     `validate:"required"`
	MaxTenure         int     `validate:"required"`
	ROI               float64 `validate:"required"`
	PreferredTenure   int     `validate:"required"`
	DateOfOffer       string  `validate:"required,IsValidDate"`
	ExpiryDateOfOffer string  `validate:"required,IsValidDate"`
	PF                float64 `validate:"ValidatePf"`
}

func ValidatePf(fl validator.FieldLevel) bool {
	Pf := fl.Field().Float()
	if utils.GetLPC() == "ANG" || utils.GetLPC() == "GRO" {
		return Pf != 0.0
	}
	return Pf == 0.0
}

func InitializeOfferColumns(row []string) (*OfferColumns, string) {

	var remarks_list []string

	float64CreditLimit, err := strconv.ParseFloat(row[2], 64)
	if err != nil {
		remarks_list = append(remarks_list, fmt.Sprintf("Error: %s", err))
		// LOGGER.Error("Error parsing CreditLimit:", err)
	}

	intMinTenure, err := strconv.Atoi(row[3])
	if err != nil {
		remarks_list = append(remarks_list, fmt.Sprintf("Error: %s", err))
		// LOGGER.Error("Error parsing MinTenure:", err)
	}

	intMaxTenure, err := strconv.Atoi(row[4])
	if err != nil {
		// LOGGER.Error("Error parsing MaxTenure:", err)
		remarks_list = append(remarks_list, fmt.Sprintf("Error: %s", err))
	}

	floatRoi, err := strconv.ParseFloat(row[5], 64)
	if err != nil {
		// LOGGER.Error("Error parsing Roi:", err)
		remarks_list = append(remarks_list, fmt.Sprintf("Error: %s", err))
	}

	intPreferredTenure, err := strconv.Atoi(row[6])
	if err != nil {
		// LOGGER.Error("Error parsing PreferredTenure:", err)
		remarks_list = append(remarks_list, fmt.Sprintf("Error: %s", err))
	}

	offerColumns := OfferColumns{
		PartnerLoanId:     row[0],
		OfferID:           row[1],
		CreditLimit:       float64CreditLimit,
		MinTenure:         intMinTenure,
		MaxTenure:         intMaxTenure,
		ROI:               floatRoi,
		PreferredTenure:   intPreferredTenure,
		DateOfOffer:       row[7],
		ExpiryDateOfOffer: row[8],
	}
	if utils.GetLPC() == "ANG" || utils.GetLPC() == "GRO" {
		float64Pf, err := strconv.ParseFloat(row[9], 64)
		if err != nil {
			// LOGGER.Error("Error parsing Pf:", err)
			remarks_list = append(remarks_list, fmt.Sprintf("Error: %s", err))
		}
		offerColumns.PF = float64Pf
	}
	return &offerColumns, strings.Join(remarks_list, ";")
}

// Validation for each row
func (f *ColumnOfferValidatorFactory) ValidateRow(row []string) (isValid bool, remarks string) {
	// Validate Number of fields in each row to be 2
	var remarks_list []string
	if len(row) != headerLength {
		return false, "Invalid number of fields present in the row"
	}
	for i := 0; i < len(row); i++ {
		if len(row[i]) == 0 {
			remarks_list = append(remarks_list, "Any field cannot be empty")
			break
		}
	}

	if len(remarks_list) != 0 {
		return false, strings.Join(remarks_list, ";")
	}

	offerColumns, initRemarks := InitializeOfferColumns(row)
	if len(initRemarks) != 0 {
		return false, initRemarks
	}

	validate := validator.New()
	validate.RegisterValidation("IsValidDate", IsValidDate)
	validate.RegisterValidation("ValidatePf", ValidatePf)

	err := validate.Struct(offerColumns)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			remarks_list = append(remarks_list, fmt.Sprintf("Field: %s, Error: %s", e.Field(), e.Tag()))
		}
	}

	return len(remarks_list) == 0, strings.Join(remarks_list, ";")
}

func (f *ColumnOfferValidatorFactory) ValidateHeader(headers []string) error {
	if utils.GetLPC() == "ANG" || utils.GetLPC() == "GRO" {
		headerLength = 10
	}
	// validate no of columns are present in the header
	if len(headers) != headerLength {
		return fmt.Errorf("invalid headers length")
	}
	if headers[0] != "partner_loan_id" {
		return fmt.Errorf("invalid header column - %s", headers[0])
	}
	if headers[1] != "offer_id" {
		return fmt.Errorf("invalid header column - %s", headers[1])
	}
	if headers[2] != "credit_limit" {
		return fmt.Errorf("invalid header column - %s", headers[2])
	}
	if headers[3] != "min_tenure" {
		return fmt.Errorf("invalid header column - %s", headers[3])
	}
	if headers[4] != "max_tenure" {
		return fmt.Errorf("invalid header column - %s", headers[4])
	}
	if headers[5] != "roi" {
		return fmt.Errorf("invalid header column - %s", headers[5])
	}
	if headers[6] != "preferred_tenure" {
		return fmt.Errorf("invalid header column - %s", headers[6])
	}
	if headers[7] != "date_of_offer" {
		return fmt.Errorf("invalid header column - %s", headers[7])
	}
	if headers[8] != "expiry_date_of_offer" {
		return fmt.Errorf("invalid header column - %s", headers[8])
	}
	if utils.GetLPC() == "ANG" || utils.GetLPC() == "GRO" {
		if headers[9] != "pf" {
			return fmt.Errorf("invalid header column - %s", headers[9])
		}
	}

	return nil
}

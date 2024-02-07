package sequentialValidator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var headerLength = 9

type OfferColumns struct {
	PartnerLoanId     string ` validate:"required"`
	OfferID           string `validate:"required"`
	CreditLimit       string ` validate:"required"`
	MinTenure         string `validate:"required"`
	MaxTenure         string `validate:"required"`
	ROI               string `validate:"required"`
	PreferredTenure   string `validate:"required"`
	DateOfOffer       string `validate:"required,IsValidDate"`
	ExpiryDateOfOffer string `validate:"required,IsValidDate"`
	PF                string `validate:"ValidatePf"`
}

func ValidatePf(fl validator.FieldLevel) bool {
	Pf := fl.Field().Float()
	if LPC == "ANG" || LPC == "GRO" {
		return Pf != 0.0
	}
	return Pf == 0.0
}

// Validation for each row
func (f *SingleOfferValidatorFactory) validateRow(row []string) (isValid bool, remarks string) {
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

	offerColumns := OfferColumns{
		PartnerLoanId:     row[0],
		OfferID:           row[1],
		CreditLimit:       row[2],
		MinTenure:         row[3],
		MaxTenure:         row[4],
		ROI:               row[5],
		PreferredTenure:   row[6],
		DateOfOffer:       row[7],
		ExpiryDateOfOffer: row[8],
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

func (f *SingleOfferValidatorFactory) validateHeader(headers []string) error {
	if LPC == "ANG" || LPC == "GRO" {
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
	if LPC == "ANG" || LPC == "GRO" {
		if headers[9] != "pf" {
			return fmt.Errorf("invalid header column - %s", headers[9])
		}
	}

	return nil
}

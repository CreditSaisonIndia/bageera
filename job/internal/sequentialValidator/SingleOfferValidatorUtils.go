package sequentialValidator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var headerLength = 9

// Validation for each row
func (f *SingleOfferValidatorFactory) validateRow(row []string) (isValid bool, remarks string) {
	// Validate Number of fields in each row to be 2
	var remarks_list []string
	if len(row) != headerLength {
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

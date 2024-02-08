package sequentialValidator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	if utils.GetLPC() == "PSB" {
		return LimitAmount == 0.0
	}
	return LimitAmount != 0.0
}

func ValidCreditLimit(fl validator.FieldLevel) bool {
	CreditLimit := fl.Field().Float()
	if utils.GetLPC() == "PSB" {
		return CreditLimit != 0.0
	}
	return CreditLimit == 0.0
}

// Validation for each row
func (f *JsonOfferValidatorFactory) validateRow(row []string) (isValid bool, remarks string) {
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

// // Header validation
// func validateHeader(headers []string) error {
// 	// validate 2 columns are present in the header
// 	if len(headers) != 2 {
// 		return fmt.Errorf("invalid headers length")
// 	}
// 	// Check if the present 2 columns are partner_loan_id and offer_details
// 	if headers[0] != "partner_loan_id" {
// 		return fmt.Errorf("invalid header column - %s", headers[0])
// 	}
// 	if headers[1] != "offer_details" {
// 		return fmt.Errorf("invalid header column - %s", headers[1])
// 	}
// 	return nil
// }

func (f *JsonOfferValidatorFactory) validateHeader(headers []string) error {
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

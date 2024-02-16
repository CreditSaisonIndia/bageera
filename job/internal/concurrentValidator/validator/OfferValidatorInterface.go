package validator

type OfferValidatorInterface interface {
	ValidateHeader(headers []string) error
	ValidateRow(row []string) (isValid bool, remarks string)
}

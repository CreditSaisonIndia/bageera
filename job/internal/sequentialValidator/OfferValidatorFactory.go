package sequentialValidator

type OfferValidatorFactory interface {
	validateHeader(headers []string) error
	validateRow(row []string) (isValid bool, remarks string)
}

type JsonOfferValidatorFactory struct{}

type ColumnOfferValidatorFactory struct{}

func GetOfferValidatorFactory(lpc string) OfferValidatorFactory {
	switch lpc {
	case "PSB", "ONL", "SPM":
		return &JsonOfferValidatorFactory{}
	default:
		return &ColumnOfferValidatorFactory{}
	}
}

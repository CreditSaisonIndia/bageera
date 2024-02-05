package sequentialValidator

type OfferValidatorFactory interface {
	validateHeader(headers []string) error
	validateRow(row []string) (isValid bool, remarks string)
}

type MultiOfferValidatorFactory struct{}

type SingleOfferValidatorFactory struct{}

func GetOfferValidatorFactory(lpc string) OfferValidatorFactory {
	switch lpc {
	case "PSB", "ONL", "SPM":
		return &MultiOfferValidatorFactory{}
	default:
		return &SingleOfferValidatorFactory{}
	}
}

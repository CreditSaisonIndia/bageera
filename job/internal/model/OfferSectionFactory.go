package model

import (
	"fmt"
	"strconv"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
)

type OfferSectionInterface interface {
	Display()
}

type CommonOfferFields struct {
	ID                string  `json:"id"`
	Interest          string  `json:"interest"`
	Tenure            int     `json:"tenure"`
	ValidTill         string  `json:"validTill"`
	PreApprovalDate   string  `json:"preApprovalDate"`
	PreferredTenure   int     `json:"preferredTenure"`
	MaxTenure         int     `json:"maxTenure"`
	MinTenure         int     `json:"minTenure"`
	ExpiryDateOfOffer string  `json:"expiryDateOfOffer"`
	CreditLimit       float64 `json:"creditLimit"`
}

type ConcreteOfferSection struct {
	CommonOfferFields
	RateOfInterest interface{} `json:"rateOfInterest"`
	Pf             interface{} `json:"pf"`
}

func (o *ConcreteOfferSection) Display() {
	fmt.Printf("ID: %s\n", o.ID)
	fmt.Printf("Interest: %s\n", o.Interest)
	fmt.Printf("Tenure: %d\n", o.Tenure)
	fmt.Printf("ValidTill: %s\n", o.ValidTill)
	fmt.Printf("PreApprovalDate: %s\n", o.PreApprovalDate)
	fmt.Printf("PreferredTenure: %d\n", o.PreferredTenure)
	fmt.Printf("MaxTenure: %d\n", o.MaxTenure)
	fmt.Printf("MinTenure: %d\n", o.MinTenure)
	fmt.Printf("ExpiryDateOfOffer: %s\n", o.ExpiryDateOfOffer)
	fmt.Printf("CreditLimit: %f\n", o.CreditLimit)
	fmt.Printf("RateOfInterest: %v\n", o.RateOfInterest)
	fmt.Printf("Pf: %v\n", o.Pf)
}

type OfferSectionFactory interface {
	CreateOfferSection(baseOffer BaseOffer) (OfferSectionInterface, error)
}

type IncOfferSectionFactory struct{}

func (f *IncOfferSectionFactory) CreateOfferSection(baseOffer BaseOffer) (OfferSectionInterface, error) {
	LOGGER := customLogger.GetLogger()
	intPreferredTenure, err := strconv.Atoi(baseOffer.(SingleCsvOffer).PreferredTenure)
	if err != nil {
		LOGGER.Error("Error parsing PreferredTenure:", err)
		return nil, err
	}

	intMinTenure, err := strconv.Atoi(baseOffer.(SingleCsvOffer).MinTenure)
	if err != nil {
		LOGGER.Error("Error parsing MinTenure:", err)
		return nil, err
	}

	intMaxTenure, err := strconv.Atoi(baseOffer.(SingleCsvOffer).MaxTenure)
	if err != nil {
		LOGGER.Error("Error parsing MaxTenure:", err)
		return nil, err
	}

	float64CreditLimit, err := strconv.ParseFloat(baseOffer.(SingleCsvOffer).CreditLimit, 64)
	if err != nil {
		LOGGER.Error("Error parsing CreditLimit:", err)
		return nil, err
	}

	floatRoi, err := strconv.ParseFloat(baseOffer.(SingleCsvOffer).Roi, 64)
	if err != nil {
		LOGGER.Error("Error parsing Roi:", err)
		return nil, err
	}
	return &ConcreteOfferSection{
		CommonOfferFields: CommonOfferFields{
			ID:                baseOffer.(SingleCsvOffer).OfferId,
			Interest:          baseOffer.(SingleCsvOffer).Roi,
			Tenure:            intPreferredTenure,
			ValidTill:         baseOffer.(SingleCsvOffer).ExpiryDateOfOffer,
			PreApprovalDate:   baseOffer.(SingleCsvOffer).DateOfOffer,
			PreferredTenure:   intPreferredTenure,
			MaxTenure:         intMaxTenure,
			MinTenure:         intMinTenure,
			ExpiryDateOfOffer: baseOffer.(SingleCsvOffer).ExpiryDateOfOffer,
			CreditLimit:       float64CreditLimit,
		},
		RateOfInterest: floatRoi, // This is an int for the second type
		Pf:             0.0,
	}, nil
}

type GROOfferSectionFactory struct{}

func (f *GROOfferSectionFactory) CreateOfferSection(baseOffer BaseOffer) (OfferSectionInterface, error) {
	LOGGER := customLogger.GetLogger()
	intPreferredTenure, err := strconv.Atoi(baseOffer.(*GroSingleCsvOffer).PreferredTenure)
	if err != nil {
		LOGGER.Error("Error parsing PreferredTenure:", err)
		return nil, err
	}

	intMinTenure, err := strconv.Atoi(baseOffer.(*GroSingleCsvOffer).MinTenure)
	if err != nil {
		LOGGER.Error("Error parsing MinTenure:", err)
		return nil, err
	}

	intMaxTenure, err := strconv.Atoi(baseOffer.(*GroSingleCsvOffer).MaxTenure)
	if err != nil {
		LOGGER.Error("Error parsing MaxTenure:", err)
		return nil, err
	}

	float64CreditLimit, err := strconv.ParseFloat(baseOffer.(*GroSingleCsvOffer).CreditLimit, 64)
	if err != nil {
		LOGGER.Error("Error parsing CreditLimit:", err)
		return nil, err
	}

	floatRoi, err := strconv.ParseFloat(baseOffer.(*GroSingleCsvOffer).Roi, 64)
	if err != nil {
		LOGGER.Error("Error parsing Roi:", err)
		return nil, err
	}

	floatPf, err := strconv.ParseFloat(baseOffer.(*GroSingleCsvOffer).Pf, 64)
	if err != nil {
		LOGGER.Error("Error parsing Roi:", err)
		return nil, err
	}
	return &ConcreteOfferSection{
		CommonOfferFields: CommonOfferFields{
			ID:                baseOffer.(*GroSingleCsvOffer).SingleCsvOffer.OfferId,
			Interest:          baseOffer.(*GroSingleCsvOffer).SingleCsvOffer.Roi,
			Tenure:            intPreferredTenure,
			ValidTill:         baseOffer.(*GroSingleCsvOffer).SingleCsvOffer.ExpiryDateOfOffer,
			PreApprovalDate:   baseOffer.(*GroSingleCsvOffer).DateOfOffer,
			PreferredTenure:   intPreferredTenure,
			MaxTenure:         intMaxTenure,
			MinTenure:         intMinTenure,
			ExpiryDateOfOffer: baseOffer.(*GroSingleCsvOffer).ExpiryDateOfOffer,
			CreditLimit:       float64CreditLimit,
		},
		RateOfInterest: floatRoi, // This is an int for the second type
		Pf:             floatPf,
	}, nil
}

func GetOfferSectionFactory(lpc string) OfferSectionFactory {
	switch lpc {
	case "GRO", "ANG":
		return &GROOfferSectionFactory{}
	default:
		return &IncOfferSectionFactory{}

	}
}

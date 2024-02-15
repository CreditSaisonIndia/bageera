package model

import (
	"fmt"
)

type OfferSection struct {
	ID                 string   `json:"id"`
	Interest           string   `json:"interest"`
	Tenure             int      `json:"tenure"`
	ValidTill          string   `json:"validTill"`
	PreApprovalDate    string   `json:"preApprovalDate"`
	PreferredTenure    int      `json:"preferredTenure"`
	MaxTenure          int      `json:"maxTenure"`
	MinTenure          int      `json:"minTenure"`
	ExpiryDateOfOffer  string   `json:"expiryDateOfOffer"`
	CreditLimit        *float64 `json:"creditLimit"`
	LimitAmount        *float64 `json:"limitAmount"`
	RateOfInterest     string   `json:"rateOfInterest"`
	PF                 float64  `json:"pf"`
	DedupeString       string   `json:"dedupeString"`
	ApplicableSegments []int    `json:"applicableSegments"`
}

type MultiOffersOfferSectionInterface interface {
	DisplayMultiOffersOfferSectionFields()
}

type MultiOffersOfferSectionCommonFields struct {
	ID                 string  `json:"id"`
	Interest           string  `json:"interest"`
	Tenure             int     `json:"tenure"`
	ValidTill          string  `json:"validTill"`
	PreApprovalDate    string  `json:"preApprovalDate"`
	PreferredTenure    int     `json:"preferredTenure"`
	MaxTenure          int     `json:"maxTenure"`
	MinTenure          int     `json:"minTenure"`
	ExpiryDateOfOffer  string  `json:"expiryDateOfOffer"`
	RateOfInterest     string  `json:"rateOfInterest"`
	PF                 float64 `json:"pf"`
	DedupeString       string  `json:"dedupeString"`
	ApplicableSegments []int   `json:"applicableSegments"`
}

type ConcreteMultiOffersOfferSection struct {
	MultiOffersOfferSectionCommonFields
	CreditLimit interface{} `json:"creditLimit"`
	LimitAmount interface{} `json:"limitAmount"`
}

func (o *ConcreteMultiOffersOfferSection) DisplayMultiOffersOfferSectionFields() {
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
}

type MultiOffersOfferSectionFactory interface {
	CreateMultiOffersOfferSection(offerInfoPointer *OfferInfo, expiryDate string, dateOfOffer string, dedupeString string) (MultiOffersOfferSectionInterface, error)
}

type PsbOfferSectionFactory struct{}

func (f *PsbOfferSectionFactory) CreateMultiOffersOfferSection(offerInfoPointer *OfferInfo, expiryDate string, dateOfOffer string, dedupeString string) (MultiOffersOfferSectionInterface, error) {
	offerInfo := *offerInfoPointer
	return &ConcreteMultiOffersOfferSection{
		MultiOffersOfferSectionCommonFields: MultiOffersOfferSectionCommonFields{
			ID:                 offerInfo.OfferID,
			Interest:           offerInfo.ROI,
			Tenure:             offerInfo.PreferredTenure,
			ValidTill:          expiryDate,
			PreApprovalDate:    dateOfOffer,
			PreferredTenure:    offerInfo.PreferredTenure,
			MaxTenure:          offerInfo.MaxTenure,
			MinTenure:          offerInfo.MinTenure,
			ExpiryDateOfOffer:  expiryDate,
			RateOfInterest:     offerInfo.ROI,
			PF:                 offerInfo.PF,
			DedupeString:       dedupeString,
			ApplicableSegments: offerInfo.ApplicableSegments,
		},
		CreditLimit: offerInfo.CreditLimit,
	}, nil
}

type OnlOfferSectionFactory struct{}

func (f *OnlOfferSectionFactory) CreateMultiOffersOfferSection(offerInfoPointer *OfferInfo, expiryDate string, dateOfOffer string, dedupeString string) (MultiOffersOfferSectionInterface, error) {
	offerInfo := *offerInfoPointer
	return &ConcreteMultiOffersOfferSection{
		MultiOffersOfferSectionCommonFields: MultiOffersOfferSectionCommonFields{
			ID:                 offerInfo.OfferID,
			Interest:           offerInfo.ROI,
			Tenure:             offerInfo.PreferredTenure,
			ValidTill:          expiryDate,
			PreApprovalDate:    dateOfOffer,
			PreferredTenure:    offerInfo.PreferredTenure,
			MaxTenure:          offerInfo.MaxTenure,
			MinTenure:          offerInfo.MinTenure,
			ExpiryDateOfOffer:  expiryDate,
			RateOfInterest:     offerInfo.ROI,
			PF:                 offerInfo.PF,
			DedupeString:       dedupeString,
			ApplicableSegments: offerInfo.ApplicableSegments,
		},
		LimitAmount: offerInfo.LimitAmount,
	}, nil
}

func GetMultiOffersOfferSectionFactory(lpc string) MultiOffersOfferSectionFactory {
	switch lpc {
	case "ONL", "SPM":
		return &OnlOfferSectionFactory{}
	default:
		return &PsbOfferSectionFactory{}

	}
}

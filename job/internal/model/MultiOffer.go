package model

type MultiCsvOffer struct {
	BaseOffer
	PartnerLoanId string        `json:"partner_loan_id"`
	OfferDetails  []OfferDetail `json:"offer_details"`
}

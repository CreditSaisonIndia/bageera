package model

type Offer struct {
	PartnerLoanID string        `json:"partner_loan_id"`
	OfferDetails  []OfferDetail `json:"offer_details"`
}

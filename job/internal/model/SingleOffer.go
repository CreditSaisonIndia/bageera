package model

type SingleCsvOffer struct {
	BaseOffer
	PartnerLoanId     string `json:"partner_loan_id"`
	OfferId           string `json:"offer_id"`
	CreditLimit       string `json:"credit_limit"`
	MinTenure         string `json:"min_tenure"`
	MaxTenure         string `json:"max_tenure"`
	Roi               string `json:"roi"`
	PreferredTenure   string `json:"preferred_tenure"`
	DateOfOffer       string `json:"date_of_offer"`
	ExpiryDateOfOffer string `json:"expiry_date_of_offer"`
}

type GroSingleCsvOffer struct {
	SingleCsvOffer
	Pf string `json:"pf"`
}

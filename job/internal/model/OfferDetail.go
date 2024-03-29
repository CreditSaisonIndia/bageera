package model

type OfferDetail struct {
	Offers            []*OfferInfo `json:"offers"`
	DateOfOffer       string       `json:"date_of_offer"`
	ExpiryDateOfOffer string       `json:"expiry_date_of_offer"`
	DedupeString      string       `json:"dedupe_string"`
}

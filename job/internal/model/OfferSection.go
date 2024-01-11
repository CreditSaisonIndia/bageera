package model

type OfferSection struct {
	ID                 string  `json:"id"`
	Interest           float64 `json:"interest"`
	Tenure             int     `json:"tenure"`
	ValidTill          string  `json:"valid_till"`
	PreApprovalDate    string  `json:"pre_approval_date"`
	PreferredTenure    int     `json:"preferred_tenure"`
	MaxTenure          int     `json:"max_tenure"`
	MinTenure          int     `json:"min_tenure"`
	ExpiryDateOfOffer  string  `json:"expiry_date_of_offer"`
	CreditLimit        float64 `json:"credit_limit"`
	RateOfInterest     float64 `json:"roi"`
	PF                 float64 `json:"pf"`
	DedupeString       int     `json:"dedupe_string"`
	ApplicableSegments []int   `json:"applicable_segments"`
}

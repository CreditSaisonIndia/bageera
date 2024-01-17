package model

type OfferSection struct {
	ID                 string  `json:"id"`
	Interest           float64 `json:"interest"`
	Tenure             int     `json:"tenure"`
	ValidTill          string  `json:"validTill"`
	PreApprovalDate    string  `json:"preApprovalDate"`
	PreferredTenure    int     `json:"preferredTenure"`
	MaxTenure          int     `json:"maxTenure"`
	MinTenure          int     `json:"minTenure"`
	ExpiryDateOfOffer  string  `json:"expiryDateOfOffer"`
	CreditLimit        float64 `json:"creditLimit"`
	RateOfInterest     float64 `json:"rateOfInterest"`
	PF                 float64 `json:"pf"`
	DedupeString       int     `json:"dedupeString"`
	ApplicableSegments []int   `json:"applicableSegments"`
}

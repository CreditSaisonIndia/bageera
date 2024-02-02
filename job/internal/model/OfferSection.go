package model

type OfferSection struct {
	ID                 string  `json:"id"`
	Interest           string  `json:"interest"`
	Tenure             int     `json:"tenure"`
	ValidTill          string  `json:"validTill"`
	PreApprovalDate    string  `json:"preApprovalDate"`
	PreferredTenure    int     `json:"preferredTenure"`
	MaxTenure          int     `json:"maxTenure"`
	MinTenure          int     `json:"minTenure"`
	ExpiryDateOfOffer  string  `json:"expiryDateOfOffer"`
	CreditLimit        float64 `json:"creditLimit"`
	LimitAmount        float64 `json:"limitAmount"`
	RateOfInterest     string  `json:"rateOfInterest"`
	PF                 float64 `json:"pf"`
	DedupeString       string  `json:"dedupeString"`
	ApplicableSegments []int   `json:"applicableSegments"`
}

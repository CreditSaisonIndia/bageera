package model

type OfferInfo struct {
	ApplicableSegments []int   `json:"applicable_segments"`
	PreferredTenure    int     `json:"preferred_tenure"`
	MaxTenure          int     `json:"max_tenure"`
	MinTenure          int     `json:"min_tenure"`
	PF                 float64 `json:"pf"`
	CreditLimit        float64 `json:"credit_limit"`
	LimitAmount        float64 `json:"limit_amount"`
	ROI                string  `json:"roi"`
	OfferID            string  `json:"offer_id"`
}

package model

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm/dialects/postgres"
)

type InitialOffer struct {
	ID                int             `json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	IsActive          bool            `json:"is_active"`
	IsDeleted         bool            `json:"is_deleted"`
	UpdatedAt         time.Time       `json:"updated_at"`
	AppFormID         sql.NullString  `json:"app_form_id"`
	PartnerLoanID     string          `json:"partner_loan_id"`
	Status            int             `json:"status"`
	OfferSections     postgres.Jsonb  `gorm:"type:jsonb" json:"offer_sections"`
	OfferRequest      json.RawMessage `gorm:"type:jsonb" json:"offer_request"`
	Description       sql.NullString  `json:"description"`
	Remarks           sql.NullString  `json:"remarks"`
	MachineError      json.RawMessage `gorm:"type:jsonb" json:"machine_error"`
	Attempt           int             `json:"attempt"`
	ExpiryDateOfOffer time.Time       `json:"expiry_date_of_offer"`
}

type Tabler interface {
	TableName() string
}

// TableName overrides the table name used by User to `profiles`
func (InitialOffer) TableName() string {
	return "initial_offer"
}

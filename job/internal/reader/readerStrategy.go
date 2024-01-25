package reader

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/model"
)

type ReaderStrategy interface {
	Read(*csv.Reader) ([]model.BaseOffer, error)
	SetHeader(header []string)
	GetHeader() []string
}

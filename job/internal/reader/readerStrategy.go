package reader

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/model"
)

/*
	Interface Reader strategy as per offers csv's
*/

type ReaderStrategy interface {
	/*
		Read method should be implemented by the concreate struct to provide a logic to read a specific use case csv
	*/
	Read(*csv.Reader) (*[]model.BaseOffer, error)

	/*
		SetHeader method should be implemented by the concreate struct to set header of offer csv
	*/
	SetHeader(header []string)

	/*
		GetHeader method should be implemented by the concreate struct to provide header of offer csv
	*/
	GetHeader() []string
}

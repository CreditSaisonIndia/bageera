package parser

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/model"
)

/*
ParserStrategy is used to parse the BaseOffer to Database Initial Offer model
*/

type ParserStrategy interface {
	/*
		parse the BaseOffer to Database Initial Offer model
	*/
	Parse(baseOffer *model.BaseOffer) (*model.InitialOffer, error)

	/*
		Writes the BaseOffer to a given csv writer
	*/
	WriteOfferToCsv(csvWriter *csv.Writer, baseOffer *model.BaseOffer)
}

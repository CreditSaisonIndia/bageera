package parserIml

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
)

/*
	Gro offer stragey written to suppport below partners
	GRO | ANG
*/

type DeleteOfferParser struct{}

// WriteOfferToCsv implements parser.Parser.
func (*DeleteOfferParser) WriteOfferToCsv(csvWriter *csv.Writer, baseOfferPointer *model.BaseOffer) {
	LOGGER := customLogger.GetLogger()
	baseOffer := *baseOfferPointer
	row := []string{
		baseOffer.(model.DeleteCsvOffer).PartnerLoanId,
	}

	if err := csvWriter.Write(row); err != nil {
		LOGGER.Error("Error while writing files to failure/success Csv:", err)
	}
}

// Parse implements parser.Parser.
func (*DeleteOfferParser) Parse(baseOfferPointer *model.BaseOffer) (*model.InitialOffer, error) {
	var proddbInitialOffer model.InitialOffer
	baseOffer := *baseOfferPointer

	proddbInitialOffer = model.InitialOffer{
		PartnerLoanID: baseOffer.(model.DeleteCsvOffer).PartnerLoanId,
	}

	return &proddbInitialOffer, nil
}

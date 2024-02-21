package parserIml

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
)

/*
	Single offer stragey written to suppport below partners
	INC | JAR | NBR
*/

type MigrateOfferParser struct{}

// WriteOfferToCsv implements parser.Parser.
func (m *MigrateOfferParser) WriteOfferToCsv(csvWriter *csv.Writer, baseOfferPointer *model.BaseOffer) {
	LOGGER := customLogger.GetLogger()
	baseOffer := *baseOfferPointer
	row := []string{
		baseOffer.(model.MigrateCsvOffer).PartnerLoanId,
	}

	if err := csvWriter.Write(row); err != nil {
		LOGGER.Error("Error while writing files to failure/success Csv:", err)
	}
}

// Parse implements parser.Parser.
func (m *MigrateOfferParser) Parse(baseOfferPointer *model.BaseOffer) (*model.InitialOffer, error) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("NO REQUIRED")
	var pointer *model.InitialOffer
	return pointer, nil
}

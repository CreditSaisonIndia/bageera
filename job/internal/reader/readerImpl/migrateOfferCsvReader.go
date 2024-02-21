package readerIml

import (
	"encoding/csv"
	"io"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
)

type MigrateOfferCsvReader struct {
	header []string
}

// GetHeader implements reader.OfferReader.
func (m *MigrateOfferCsvReader) GetHeader() []string {
	return m.header
}

// SetHeader implements reader.OfferReader.
func (m *MigrateOfferCsvReader) SetHeader(header []string) {
	m.header = header
}

// Read implements reader.OfferReader.
func (*MigrateOfferCsvReader) Read(csvReader *csv.Reader) (*[]model.BaseOffer, error) {
	LOGGER := customLogger.GetLogger()
	var baseOfferArr []model.BaseOffer

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		migrateCsvOffer := model.MigrateCsvOffer{
			PartnerLoanId: record[0],
		}

		baseOfferArr = append(baseOfferArr, migrateCsvOffer)
	}

	LOGGER.Info("Chunk 1st parnterLoanId -->>  ", baseOfferArr[0].(model.MigrateCsvOffer).PartnerLoanId)
	LOGGER.Info("Chunk Last parnterLoanId -->>  " + baseOfferArr[len(baseOfferArr)-1].(model.MigrateCsvOffer).PartnerLoanId)

	return &baseOfferArr, nil
}

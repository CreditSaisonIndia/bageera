package readerIml

import (
	"encoding/csv"
	"io"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
)

type DeleteOfferCsvReader struct {
	header []string
}

// GetHeader implements reader.OfferReader.
func (m *DeleteOfferCsvReader) GetHeader() []string {
	return m.header
}

// SetHeader implements reader.OfferReader.
func (m *DeleteOfferCsvReader) SetHeader(header []string) {
	m.header = header
}

// Read implements reader.OfferReader.
func (*DeleteOfferCsvReader) Read(csvReader *csv.Reader) (*[]model.BaseOffer, error) {
	LOGGER := customLogger.GetLogger()
	var baseOfferArr []model.BaseOffer

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		deleteCsvOffer := model.DeleteCsvOffer{
			PartnerLoanId: record[0],
		}

		baseOfferArr = append(baseOfferArr, deleteCsvOffer)
	}

	LOGGER.Info("Chunk 1st parnterLoanId -->>  ", baseOfferArr[0].(model.DeleteCsvOffer).PartnerLoanId)
	LOGGER.Info("Chunk Last parnterLoanId -->>  " + baseOfferArr[len(baseOfferArr)-1].(model.DeleteCsvOffer).PartnerLoanId)

	return &baseOfferArr, nil
}

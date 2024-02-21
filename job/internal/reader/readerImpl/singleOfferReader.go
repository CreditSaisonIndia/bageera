package readerIml

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
)

type SingleOfferReader struct {
	header []string
}

// GetHeader implements reader.OfferReader.
func (m *SingleOfferReader) GetHeader() []string {
	return m.header
}

// SetHeader implements reader.OfferReader.
func (m *SingleOfferReader) SetHeader(header []string) {
	m.header = header
}

// Read implements reader.OfferReader.
func (*SingleOfferReader) Read(csvReader *csv.Reader) (*[]model.BaseOffer, error) {
	LOGGER := customLogger.GetLogger()
	var baseOfferArr []model.BaseOffer

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		singleCsvOffer := model.SingleCsvOffer{
			PartnerLoanId:     record[0],
			OfferId:           record[1],
			CreditLimit:       record[2],
			MinTenure:         record[3],
			MaxTenure:         record[4],
			Roi:               record[5],
			PreferredTenure:   record[6],
			DateOfOffer:       record[7],
			ExpiryDateOfOffer: record[8],
		}

		baseOfferArr = append(baseOfferArr, singleCsvOffer)
	}
	offersLen := len(baseOfferArr)
	if offersLen > 0 {
		LOGGER.Info("Chunk 1st parnterLoanId -->>  ", baseOfferArr[0].(model.SingleCsvOffer).PartnerLoanId)
		LOGGER.Info("Chunk Last parnterLoanId -->>  " + baseOfferArr[offersLen-1].(model.SingleCsvOffer).PartnerLoanId)
	} else {
		LOGGER.Error("NO OFFER FOUND IN")
		return nil, fmt.Errorf("NO OFFER LEFT TO READ")
	}

	return &baseOfferArr, nil
}

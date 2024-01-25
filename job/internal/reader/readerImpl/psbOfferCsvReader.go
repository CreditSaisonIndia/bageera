package readerIml

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
)

type PsbOfferCsvReader struct {
	header []string
}

// GetHeader implements reader.OfferReader.
func (m *PsbOfferCsvReader) GetHeader() []string {
	return m.header
}

// SetHeader implements reader.OfferReader.
func (m *PsbOfferCsvReader) SetHeader(header []string) {
	m.header = []string{"partner_loan_id", "offer_details"}
}

func (r *PsbOfferCsvReader) Read(csvReader *csv.Reader) ([]model.BaseOffer, error) {
	LOGGER := customLogger.GetLogger()
	var baseOfferArr []model.BaseOffer

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		var offerDetails []model.OfferDetail
		if err := json.Unmarshal([]byte(record[1]), &offerDetails); err != nil {
			LOGGER.Error(err)
			return nil, err
		}

		multiCsvOffer := model.MultiCsvOffer{
			PartnerLoanId: record[0],
			OfferDetails:  offerDetails,
		}

		baseOfferArr = append(baseOfferArr, &multiCsvOffer)
	}
	log.Println(utils.PrettyPrint(baseOfferArr))
	LOGGER.Info("Chunk 1st parnterLoanId -->>  ", baseOfferArr[0].(*model.MultiCsvOffer).PartnerLoanId)
	LOGGER.Info("Chunk Last parnterLoanId -->>  " + baseOfferArr[len(baseOfferArr)-1].(*model.MultiCsvOffer).PartnerLoanId)

	return baseOfferArr, nil
}

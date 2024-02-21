package reader

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/model"
	readerIml "github.com/CreditSaisonIndia/bageera/internal/reader/readerImpl"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

type Reader struct {
	readerStrategy ReaderStrategy
}

func SetReader(readerStrategy ReaderStrategy) *Reader {
	return &Reader{
		readerStrategy: readerStrategy,
	}
}

func (r *Reader) ReaderStrategy(csvReader *csv.Reader) (*[]model.BaseOffer, error) {
	return r.readerStrategy.Read(csvReader)
}

func (r *Reader) GetHeader() []string {
	return r.readerStrategy.GetHeader()
}

// SetHeader implements reader.OfferReader.
func (r *Reader) SetHeader(header []string) {
	r.readerStrategy.SetHeader(header)
}

func GetReaderType(csvReader *csv.Reader) *Reader {

	switch serviceConfig.ApplicationSetting.Lpc {
	case "PSB", "ONL", "SPM":
		multiOfferCsvReader := &readerIml.MultiOfferCsvReader{}

		return SetReader(multiOfferCsvReader)

	case "GRO", "ANG":
		groCsvOfferReader := &readerIml.GroCsvOfferReader{}

		return SetReader(groCsvOfferReader)

	case "migrate":
		migrateCsvOfferReader := &readerIml.MigrateOfferCsvReader{}

		return SetReader(migrateCsvOfferReader)

	default:
		singleOfferReader := &readerIml.SingleOfferReader{}

		return SetReader(singleOfferReader)
	}
}

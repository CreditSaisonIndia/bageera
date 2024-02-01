package reader

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/model"
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

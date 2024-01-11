package csvUtilityWrapper

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/CreditSaisonIndia/bageera/internal/model"
)

func NewCSVReader(reader io.Reader) *csv.Reader {
	return csv.NewReader(reader)
}

func WriteOfferToCSV(csvWriter *csv.Writer, offer model.Offer) {

	offerDetailsString, err := json.Marshal(offer.OfferDetails)
	if err != nil {
		log.Println("Error while marshalling to json b:", err)
		os.Exit(1)
	}
	row := []string{
		offer.PartnerLoanID, string(offerDetailsString),
	}

	if err := csvWriter.Write(row); err != nil {
		log.Println("Error writing offer to CSV:", err)
	}
}

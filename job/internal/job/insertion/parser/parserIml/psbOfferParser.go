package parserIml

import (
	"encoding/csv"
	"encoding/json"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/utils"
	"github.com/jinzhu/gorm/dialects/postgres"
)

type PsbOfferParser struct{}

// WriteOfferToCsv implements parser.Parser.
func (*PsbOfferParser) WriteOfferToCsv(csvWriter *csv.Writer, baseOffer model.BaseOffer) {
	LOGGER := customLogger.GetLogger()
	offerDetailsString, err := json.Marshal(baseOffer.(*model.MultiCsvOffer).OfferDetails)
	if err != nil {
		LOGGER.Error("Error while marshalling offer details while writing files to failure/success Csv:", err)
	}
	row := []string{
		baseOffer.(*model.MultiCsvOffer).PartnerLoanId, string(offerDetailsString),
	}

	if err := csvWriter.Write(row); err != nil {
		LOGGER.Error("Error writing offer to failure/success Csv:", err)
	}
}

// GetFileHeader implements parser.Parser.

// parse implements parser.Parser.
func (*PsbOfferParser) Parse(baseOffer model.BaseOffer) (model.InitialOffer, error) {
	LOGGER := customLogger.GetLogger()
	var proddbInitialOffer model.InitialOffer
	expiryDateOfOffer, rawSection, err := getSection(baseOffer.(*model.MultiCsvOffer).OfferDetails)
	if err != nil {
		LOGGER.Info("Error While forming marshalling section with following error : ", err)
		return proddbInitialOffer, err
	}
	proddbInitialOffer = model.InitialOffer{
		// Set other fields based on your requirements
		IsActive:          true, // For example
		IsDeleted:         false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		PartnerLoanID:     baseOffer.(*model.MultiCsvOffer).PartnerLoanId,
		Status:            30,
		OfferSections:     postgres.Jsonb{rawSection},
		ExpiryDateOfOffer: expiryDateOfOffer,

		// ApplicableSegments: firstOffer.ApplicableSegments,
		// Set other fields as needed
	}

	return proddbInitialOffer, nil
}

func getSection(offers []model.OfferDetail) (time.Time, []byte, error) {

	LOGGER := customLogger.GetLogger()
	var offerSectionArray []model.OfferSection
	var expiryDateOfOffer time.Time

	for _, resp := range offers {
		var internalOfferArray = resp.Offers

		for _, internalOffer := range internalOfferArray {
			println(utils.PrettyPrint(internalOffer))
			section := model.OfferSection{
				ID:                 internalOffer.OfferID,
				Interest:           internalOffer.ROI,
				Tenure:             internalOffer.PreferredTenure,
				ValidTill:          resp.ExpiryDateOfOffer,
				PreApprovalDate:    resp.DateOfOffer,
				PreferredTenure:    internalOffer.PreferredTenure,
				MaxTenure:          internalOffer.MaxTenure,
				MinTenure:          internalOffer.MinTenure,
				ExpiryDateOfOffer:  resp.ExpiryDateOfOffer,
				CreditLimit:        internalOffer.CreditLimit,
				LimitAmount:        internalOffer.LimitAmount,
				RateOfInterest:     internalOffer.ROI,
				PF:                 internalOffer.PF,
				DedupeString:       resp.DedupeString,
				ApplicableSegments: internalOffer.ApplicableSegments,
			}
			offerSectionArray = append(offerSectionArray, section)

		}
		layout := "2006-01-02"
		expiryDate, err := time.Parse(layout, resp.ExpiryDateOfOffer)
		if err != nil {
			LOGGER.Error("Error parsing date:", err)
			return time.Now(), nil, err
		}
		expiryDateOfOffer = expiryDate

	}

	rawMessage, err := json.Marshal(offerSectionArray)
	if err != nil {
		LOGGER.Info("Error while marshalling to json b:", err)
		return time.Now(), nil, err
	}

	// Print the json.RawMessage
	return expiryDateOfOffer, rawMessage, nil
}

package parserIml

import (
	"encoding/csv"
	"encoding/json"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/jinzhu/gorm/dialects/postgres"
)

/*
	Single offer stragey written to suppport below partners
	INC | JAR | NBR
*/

type SingleOfferParser struct{}

// WriteOfferToCsv implements parser.Parser.
func (*SingleOfferParser) WriteOfferToCsv(csvWriter *csv.Writer, baseOfferPointer *model.BaseOffer) {
	LOGGER := customLogger.GetLogger()
	baseOffer := *baseOfferPointer
	row := []string{
		baseOffer.(model.SingleCsvOffer).PartnerLoanId,
		baseOffer.(model.SingleCsvOffer).OfferId,
		baseOffer.(model.SingleCsvOffer).CreditLimit,
		baseOffer.(model.SingleCsvOffer).MinTenure,
		baseOffer.(model.SingleCsvOffer).MaxTenure,
		baseOffer.(model.SingleCsvOffer).Roi,
		baseOffer.(model.SingleCsvOffer).PreferredTenure,
		baseOffer.(model.SingleCsvOffer).DateOfOffer,
		baseOffer.(model.SingleCsvOffer).ExpiryDateOfOffer,
	}

	if err := csvWriter.Write(row); err != nil {
		LOGGER.Error("Error while writing files to failure/success Csv:", err)
	}
}

// Parse implements parser.Parser.
func (*SingleOfferParser) Parse(baseOfferPointer *model.BaseOffer) (*model.InitialOffer, error) {
	LOGGER := customLogger.GetLogger()
	var proddbInitialOffer model.InitialOffer
	baseOffer := *baseOfferPointer
	expiryDateOfOffer, rawSection, err := getSectionForSingleOffer(baseOffer)
	if err != nil {
		LOGGER.Info("Error While forming marshalling section with following error : ", err)
		return &proddbInitialOffer, err
	}
	proddbInitialOffer = model.InitialOffer{
		// Set other fields based on your requirements
		IsActive:      true, // For example
		IsDeleted:     false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		PartnerLoanID: baseOffer.(model.SingleCsvOffer).PartnerLoanId,
		Status:        30,
		OfferSections: postgres.Jsonb{rawSection},
		ExpiryDate:    expiryDateOfOffer,

		// ApplicableSegments: firstOffer.ApplicableSegments,
		// Set other fields as needed
	}

	return &proddbInitialOffer, nil
}

func getSectionForSingleOffer(offer model.BaseOffer) (time.Time, []byte, error) {

	LOGGER := customLogger.GetLogger()
	var expiryDateOfOffer time.Time
	var offerSectionArray []model.OfferSectionInterface

	sectionFactory := model.GetOfferSectionFactory(serviceConfig.ApplicationSetting.Lpc)
	section, err := sectionFactory.CreateOfferSection(offer)
	if err != nil {
		LOGGER.Error("Error parsing incFactory.CreateOfferSection:", err)
		return time.Now(), nil, err
	}

	layout := "2006-01-02"
	expiryDate, err := time.Parse(layout, offer.(model.SingleCsvOffer).ExpiryDateOfOffer)
	if err != nil {
		LOGGER.Error("Error parsing date:", err)
		return time.Now(), nil, err
	}
	expiryDateOfOffer = expiryDate
	offerSectionArray = append(offerSectionArray, section)
	rawMessage, err := json.Marshal(offerSectionArray)
	if err != nil {
		LOGGER.Info("Error while marshalling to json b:", err)
		return time.Now(), nil, err
	}

	// Print the json.RawMessage
	return expiryDateOfOffer, rawMessage, nil
}

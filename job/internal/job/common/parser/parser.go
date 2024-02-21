package parser

import (
	"encoding/csv"
	"strconv"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/job/common/parser/parserIml"
	"github.com/CreditSaisonIndia/bageera/internal/model"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

type Parser struct {
	parserStrategy ParserStrategy
}

func SetParser(parserStrategy ParserStrategy) *Parser {
	return &Parser{
		parserStrategy: parserStrategy,
	}
}

func (p *Parser) ParserStrategy(baseOffer *model.BaseOffer) (*model.InitialOffer, error) {
	return p.parserStrategy.Parse(baseOffer)
}

func (p *Parser) WriteOfferToCsv(csvWriter *csv.Writer, baseOffer *model.BaseOffer) {
	p.parserStrategy.WriteOfferToCsv(csvWriter, baseOffer)
}

func (p *Parser) WriteInitialOfferToCsv(csvWriter *csv.Writer, initialOfferPointer *model.InitialOffer) {
	p.WriteInitialOfferToCsvLocal(csvWriter, initialOfferPointer)
}

func (p *Parser) WriteInitialOfferToCsvLocal(csvWriter *csv.Writer, initialOfferPointer *model.InitialOffer) {
	LOGGER := customLogger.GetLogger()
	initialOffer := *initialOfferPointer
	row := []string{
		strconv.Itoa(initialOffer.ID),                        // Convert ID to string
		initialOffer.CreatedAt.Format("2006-01-02 15:04:05"), // Convert CreatedAt to string with format
		strconv.FormatBool(initialOffer.IsActive),            // Convert IsActive to string
		strconv.FormatBool(initialOffer.IsDeleted),           // Convert IsDeleted to string
		initialOffer.UpdatedAt.Format("2006-01-02 15:04:05"), // Convert UpdatedAt to string with format
		initialOffer.AppFormID.String,                        // AppFormID is already a string, no need for conversion
		initialOffer.PartnerLoanID,
		strconv.Itoa(initialOffer.Status),             // Convert Status to string
		string(initialOffer.OfferSections.RawMessage), // Convert OfferSections to string
		string(initialOffer.OfferRequest),             // Convert OfferRequest to string
		initialOffer.Description.String,
		initialOffer.Remarks.String,
		string(initialOffer.MachineError),
		strconv.Itoa(initialOffer.Attempt),                    // Convert Attempt to string
		initialOffer.ExpiryDate.Format("2006-01-02 15:04:05"), // Convert ExpiryDate to string with format

	}

	if err := csvWriter.Write(row); err != nil {
		LOGGER.Error("Error while writing files to failure/success Csv:", err)
	}
}

func GetParserType() *Parser {

	switch serviceConfig.ApplicationSetting.Lpc {
	case "PSB", "ONL", "SPM":
		multiOfferParser := &parserIml.MultiOfferParser{}
		return SetParser(multiOfferParser)
	case "GRO", "ANG":
		groOfferParser := &parserIml.GroOfferParser{}
		return SetParser(groOfferParser)

	case "migrate":
		migrateOfferParser := &parserIml.MigrateOfferParser{}
		return SetParser(migrateOfferParser)

	default: //JAR | NBR | INC
		singleOfferParser := &parserIml.SingleOfferParser{}
		return SetParser(singleOfferParser)
	}

}

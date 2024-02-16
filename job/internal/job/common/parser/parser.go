package parser

import (
	"encoding/csv"

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

func GetParserType() *Parser {

	switch serviceConfig.ApplicationSetting.Lpc {
	case "PSB", "ONL", "SPM":
		multiOfferParser := &parserIml.MultiOfferParser{}
		return SetParser(multiOfferParser)
	case "GRO", "ANG":
		groOfferParser := &parserIml.GroOfferParser{}
		return SetParser(groOfferParser)

	default: //JAR | NBR | INC
		singleOfferParser := &parserIml.SingleOfferParser{}
		return SetParser(singleOfferParser)
	}

}

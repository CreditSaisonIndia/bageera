package parser

import (
	"encoding/csv"

	"github.com/CreditSaisonIndia/bageera/internal/model"
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

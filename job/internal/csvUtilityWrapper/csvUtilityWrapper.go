package csvUtilityWrapper

import (
	"encoding/csv"
	"io"
)

func NewCSVReader(reader io.Reader) *csv.Reader {
	return csv.NewReader(reader)
}

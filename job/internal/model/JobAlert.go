package model

type BaseAlert struct {
	FileName string `json:"fileName"`
	Lpc      string `json:"lpc"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

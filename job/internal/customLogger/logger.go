package customLogger

import (
	"encoding/json"

	"go.uber.org/zap"
)

var LOGGER *zap.SugaredLogger

func GetLogger() *zap.SugaredLogger {
	return LOGGER
}

func InitLogger() {

	var cfg *zap.Config

	loggerConfig := []byte(`{
		"level": "info",
		"encoding": "console",
		"timekey": "timestamp",
		"outputPaths": ["stdout", "/tmp/logs"],
		"errorOutputPaths": ["stderr"],
		"encoderConfig": {
		  "messageKey": "message",
		  "levelKey": "level",
		  "levelEncoder": "lowercase"
		}
	  }`)

	if err := json.Unmarshal(loggerConfig, &cfg); err != nil {
		panic(err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	LOGGER = logger.Sugar()
}

package main

import (
	"os"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/queueConsumer"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
)

func main() {

	customLogger.InitLogger()
	LOGGER := customLogger.GetLogger()

	env := os.Getenv("environment")

	LOGGER.Info("Reading and Setting up Server,Database and Application Level Properties.")
	serviceConfig.SetUp(env)
	LOGGER.Info("Properties configuration successful.")

	retries := 3

	for i := 0; i < retries; i++ {
		LOGGER.Info("****** Running Queue consumer *******")
		if err := queueConsumer.Consume(); err != nil {
			LOGGER.Info("Error while running runQueueConsumer | hence retrying in 10 sec :", err)
			time.Sleep(10 * time.Second)
			continue
		}
		// You may also want to introduce a delay between consecutive runs
		LOGGER.Info("******RETURN  FROM Queue consumer*******")
		time.Sleep(5 * time.Second)
		LOGGER.Info("Retrying...")
	}
	LOGGER.Info("**** JOB IS DONE | KILLING THE INSTANCE ****")
}

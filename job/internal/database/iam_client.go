package database

import (
	"context"
	"fmt"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"go.uber.org/zap"
)

type IAM struct {
	LOGGER *zap.SugaredLogger
}

func NewIAMClient() *IAM {
	return &IAM{}
}

func (iamHandler *IAM) GetIamRdsCredential(ctx context.Context, host string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		iamHandler.LOGGER.Info("Could not create config for pgxpoll. Exiting ...")
		return "", err
	}

	// 840000 milliseconds = 14 minutes
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 840000*time.Millisecond)
	defer cancel()

	var dbUser string = serviceConfig.DatabaseSetting.User
	var dbHost string = host
	var dbPort string = serviceConfig.DatabaseSetting.Port
	var dbEndpoint string = fmt.Sprintf("%s:%s", dbHost, dbPort)
	var region string = serviceConfig.ApplicationSetting.Region

	authenticationToken, err := auth.BuildAuthToken(ctxWithTimeout, dbEndpoint, region, dbUser, cfg.Credentials)
	if err != nil {
		iamHandler.LOGGER.Info("Could not retrieve authenticationToken from AWS. Exiting ...")
		return "", err
	}

	return authenticationToken, nil
}

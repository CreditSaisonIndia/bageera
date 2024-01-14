package database

import (
	"context"
	"database/sql"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func GetDb() *gorm.DB {
	return db
}

func InitGormDb() (*gorm.DB, error) {

	LOGGER := customLogger.GetLogger()

	var sqlxMasterDb *sql.DB
	var err error

	// First create master node connection
	if serviceConfig.ApplicationSetting.RunType == "local" {
		LOGGER.Info("Using Local Database Connection [MASTER_NODE].")

		// Information for local connection
		localAuth := &LocalAuth{
			DatabaseUser:     serviceConfig.DatabaseSetting.User,
			DatabaseHost:     serviceConfig.DatabaseSetting.MasterDbHost,
			DatabasePort:     serviceConfig.DatabaseSetting.Port,
			DatabaseName:     serviceConfig.DatabaseSetting.Name,
			DatabasePassword: serviceConfig.DatabaseSetting.Password,
			DatabaseSchema:   serviceConfig.DatabaseSetting.TablePrefix,
		}

		sqlxMasterDb, err = localAuth.Connect(context.TODO())
		if err != nil {
			LOGGER.Error("Unable to create connection to database. Exiting ...")
			return nil, err
		}

	} else {
		// If the RunType is cloud or any other, we will get a connection to given RDS instance
		// using IAM user auth approach
		LOGGER.Info("Using RDS Database Connection [MASTER_NODE].")

		// Information for IAM Authentication RDS connection
		iamAuth := &IAMAuth{
			DatabaseUser:   serviceConfig.DatabaseSetting.User,
			DatabaseHost:   serviceConfig.DatabaseSetting.MasterDbHost,
			DatabasePort:   serviceConfig.DatabaseSetting.Port,
			DatabaseName:   serviceConfig.DatabaseSetting.Name,
			DatabaseSchema: serviceConfig.DatabaseSetting.TablePrefix,
		}

		sqlxMasterDb, err = iamAuth.Connect(context.TODO())
		if err != nil {
			LOGGER.Error("Unable to create connection to database. Exiting ...")
			return nil, err
		}
	}

	gormDB, err := gorm.Open(gormPostgres.New(gormPostgres.Config{
		Conn: sqlxMasterDb,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	if err != nil {
		LOGGER.Error("Unable to connect to the database. Exiting ...")
		return nil, err
	}

	LOGGER.Info("Application successfully connected to RDS Scarlet database.")

	// Initialize db for application
	return gormDB, nil

}

func SetDb() {
	gormLocal, _ := InitGormDb()
	db = gormLocal
}

var sqlDb *sql.DB

func GetSqlDb() *sql.DB {
	return sqlDb
}

func InitSqlxDb() {

	LOGGER := customLogger.GetLogger()

	var sqlxMasterDb *sql.DB
	var err error

	// First create master node connection
	if serviceConfig.ApplicationSetting.RunType == "local" {
		LOGGER.Info("Using Local Database Connection [MASTER_NODE].")

		// Information for local connection
		localAuth := &LocalAuth{
			DatabaseUser:     serviceConfig.DatabaseSetting.User,
			DatabaseHost:     serviceConfig.DatabaseSetting.MasterDbHost,
			DatabasePort:     serviceConfig.DatabaseSetting.Port,
			DatabaseName:     serviceConfig.DatabaseSetting.Name,
			DatabasePassword: serviceConfig.DatabaseSetting.Password,
			DatabaseSchema:   serviceConfig.DatabaseSetting.TablePrefix,
		}

		sqlxMasterDb, err = localAuth.Connect(context.TODO())
		if err != nil {
			LOGGER.Error("Unable to create connection to database. Exiting ...")
			panic(err)
		}

	} else {
		// If the RunType is cloud or any other, we will get a connection to given RDS instance
		// using IAM user auth approach
		LOGGER.Info("Using RDS Database Connection [MASTER_NODE].")

		// Information for IAM Authentication RDS connection
		iamAuth := &IAMAuth{
			DatabaseUser:   serviceConfig.DatabaseSetting.User,
			DatabaseHost:   serviceConfig.DatabaseSetting.MasterDbHost,
			DatabasePort:   serviceConfig.DatabaseSetting.Port,
			DatabaseName:   serviceConfig.DatabaseSetting.Name,
			DatabaseSchema: serviceConfig.DatabaseSetting.TablePrefix,
		}

		sqlxMasterDb, err = iamAuth.Connect(context.TODO())
		if err != nil {
			LOGGER.Error("Unable to create connection to database. Exiting ...")
			panic(err)
		}
	}

	LOGGER.Info("Application successfully connected to RDS Scarlet database.")

	// Initialize db for application
	sqlDb = sqlxMasterDb

}

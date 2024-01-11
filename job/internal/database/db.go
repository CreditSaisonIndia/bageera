package database

import (
	"context"
	"database/sql"
	"net"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

var db *gorm.DB

func GetDb() *gorm.DB {
	return db
}

func InitGormPool() {

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

	gormDB, err := gorm.Open(gormPostgres.New(gormPostgres.Config{
		Conn: sqlxMasterDb,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	// If the RunType is local, then we will connect the application to local database or the
	// tunneled database
	if serviceConfig.ApplicationSetting.RunType == "local" {
		LOGGER.Info("Using Local Database Connection [READER_NODE].")
		readerLocalAuth := &LocalAuth{
			DatabaseUser:   serviceConfig.DatabaseSetting.User,
			DatabaseHost:   serviceConfig.DatabaseSetting.ReaderDbHost,
			DatabasePort:   serviceConfig.DatabaseSetting.Port,
			DatabaseName:   serviceConfig.DatabaseSetting.Name,
			DatabaseSchema: serviceConfig.DatabaseSetting.TablePrefix,
		}

		gormDB.Use(dbresolver.Register(dbresolver.Config{
			Replicas: []gorm.Dialector{
				gormPostgres.New(gormPostgres.Config{
					DSN:                  readerLocalAuth.GetLocalReaderNodeConnectionString(context.TODO(), net.LookupCNAME),
					PreferSimpleProtocol: true,
				}),
			},
			TraceResolverMode: true,
		}))

	} else {
		LOGGER.Info("Using RDS Database Connection")
		readerIamAuth := &IAMAuth{
			DatabaseUser:   serviceConfig.DatabaseSetting.User,
			DatabaseHost:   serviceConfig.DatabaseSetting.MasterDbHost,
			DatabasePort:   serviceConfig.DatabaseSetting.Port,
			DatabaseName:   serviceConfig.DatabaseSetting.Name,
			DatabaseSchema: serviceConfig.DatabaseSetting.TablePrefix,
		}
		readerDbConnection, err := readerIamAuth.Connect(context.TODO())
		if err != nil {
			LOGGER.Error("Unable to create connection to database. Exiting ...")
			panic(err)
		}

		gormDB.Use(dbresolver.Register(dbresolver.Config{
			Replicas: []gorm.Dialector{
				gormPostgres.New(gormPostgres.Config{
					Conn:                 readerDbConnection,
					PreferSimpleProtocol: true,
				}),
			},
			TraceResolverMode: true,
		}).SetConnMaxIdleTime(1 * time.Minute).
			SetConnMaxLifetime(13 * time.Minute).
			SetMaxIdleConns(2).SetMaxOpenConns(4))
	}

	if err != nil {
		LOGGER.Error("Unable to connect to the database. Exiting ...")
		panic(err)
	}

	LOGGER.Info("Application successfully connected to RDS Scarlet database.")

	// Initialize db for application
	db = gormDB
}

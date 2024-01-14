package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/CreditSaisonIndia/bageera/internal/serviceConfig"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/jmoiron/sqlx"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDb() *sql.DB {
	//dbUsername := config.Get("dbUsername")
	//dbPassword := config.Get("dbPassword")
	//dbHost := config.Get("dbHost")
	//dbPort := config.Get("dbPort")
	//dbName := config.Get("dbName")
	//schema := config.Get("schema")
	dbUsername := "myuser"
	dbPassword := "password"
	dbHost := "localhost"
	dbPort := "5432"
	dbName := "mydb"
	schema := "public"
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", dbUsername, dbPassword, dbHost, dbPort, dbName, schema)
	log.Printf("connectionString : ", connectionString)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal("Error while opening connection to data base : ", err)
	}
	// Set maximum idle connections
	db.SetMaxIdleConns(10)
	// Set maximum open connections
	db.SetMaxOpenConns(100)
	err = db.Ping()

	return db
}

type IamAuth struct {
	DatabaseUser       string
	DatabaseHost       string
	DatabasePort       string
	DatabaseName       string
	AmazonResourceName string
	AuthTokenGenerator Generator
	DatabaseSchema     string
}

var testDB *gorm.DB

func GetTestDb() *gorm.DB {
	return db
}

func Open() {
	LOGGER := customLogger.GetLogger()
	iamAuth := &IamAuth{
		DatabaseUser:   serviceConfig.DatabaseSetting.User,
		DatabaseHost:   serviceConfig.DatabaseSetting.MasterDbHost,
		DatabasePort:   serviceConfig.DatabaseSetting.Port,
		DatabaseName:   serviceConfig.DatabaseSetting.Name,
		DatabaseSchema: serviceConfig.DatabaseSetting.TablePrefix,
	}

	sess := session.Must(session.NewSession())
	fmt.Println("Generating IAM Auth Token for user: ", iamAuth.DatabaseUser)

	authToken, err := rdsutils.BuildAuthToken(iamAuth.DatabaseHost, serviceConfig.ApplicationSetting.Region, iamAuth.DatabaseUser, sess.Config.Credentials)
	if err != nil {
		panic("failed to create authentication token: " + err.Error())
	}
	dbURI := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s search_path=scarlet sslmode=require",
		iamAuth.DatabaseHost, iamAuth.DatabasePort, iamAuth.DatabaseUser, iamAuth.DatabaseName, authToken) //Build connection string
	fmt.Println("auth_token : ", authToken)
	fmt.Println("db_uri : ", dbURI)

	// Initialize a new db connection.
	log.Println("connecting to db")
	sqlxMasterDb, err := sqlx.Connect("postgres", dbURI)

	if err != nil {
		log.Println("error", err.Error())
		panic(err)
	} else {
		log.Println("connected")
	}

	gormDB, err := gorm.Open(gormPostgres.New(gormPostgres.Config{
		Conn: sqlxMasterDb,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})

	if err != nil {
		LOGGER.Error("Unable to connect to the database. Exiting ...")
		panic(err)
	}

	LOGGER.Info("Application successfully connected to RDS Scarlet database.")

	// Initialize db for application
	testDB = gormDB
}

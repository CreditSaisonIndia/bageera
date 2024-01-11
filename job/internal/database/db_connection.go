package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/CreditSaisonIndia/bageera/internal/customLogger"
	"github.com/jackc/pgx/v4/stdlib"
	"golang.org/x/xerrors"
)

type IAMAuth struct {
	DatabaseUser       string
	DatabaseHost       string
	DatabasePort       string
	DatabaseName       string
	AmazonResourceName string
	AuthTokenGenerator Generator
	DatabaseSchema     string
}

type LocalAuth struct {
	DatabaseUser     string
	DatabaseHost     string
	DatabasePort     string
	DatabaseName     string
	DatabasePassword string
	DatabaseSchema   string
}

type iamDB struct {
	IAMAuth
}

type localDB struct {
	LocalAuth
}

type IAMAuthGenerator struct{}

type Generator interface {
	GetAuthToken(ctx context.Context) (string, error)
}

type LookupCNAME func(string) (string, error)

func (ia *IAMAuth) GetConnectionString(ctx context.Context, lookup LookupCNAME) (string, error) {

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("GetConnectionString")
	authenticationToken, err := NewIAMClient().GetIamRdsCredential(ctx, ia.DatabaseHost)

	if err != nil {
		LOGGER.Error("XXXX-XXXX-XXXX-XXXX Failed to retrieve authenticationToken for establishing a new connection.")
		return "", xerrors.Errorf("could not build auth token: %w", err)
	} else {
		LOGGER.Info("XXXX-XXXX-XXXX-XXXX Successfully retrieved authenticationToken for establishing a new connection.")
	}

	var postgresConnection strings.Builder

	postgresConnection.WriteString(
		fmt.Sprintf("user=%s dbname=%s sslmode=%s port=%s host=%s password=%s search_path=%s",
			ia.DatabaseUser,
			ia.DatabaseName,
			"require",
			ia.DatabasePort,
			ia.DatabaseHost,
			authenticationToken, ia.DatabaseSchema))

	return postgresConnection.String(), nil
}

func (la *LocalAuth) GetLocalReaderNodeConnectionString(ctx context.Context, lookup LookupCNAME) string {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("GetLocalReaderNodeConnectionString")
	var postgresConnection strings.Builder

	LOGGER.Info("Building Connection String for Local Postgres DB")
	postgresConnection.WriteString(
		fmt.Sprintf("user=%s dbname=%s sslmode=%s port=%s host=%s password=%s search_path=%s",
			la.DatabaseUser,
			la.DatabaseName,
			"disable",
			la.DatabasePort,
			la.DatabaseHost,
			la.DatabasePassword, la.DatabaseSchema))

	LOGGER.Info("Connection String ", postgresConnection.String())
	return postgresConnection.String()
}

func (la *LocalAuth) GetConnectionString(ctx context.Context, lookup LookupCNAME) (string, error) {

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("GetConnectionString")
	var postgresConnection strings.Builder

	LOGGER.Info("Building Connection String for Local Postgres DB")
	postgresConnection.WriteString(
		fmt.Sprintf("user=%s dbname=%s sslmode=%s port=%s host=%s password=%s search_path=%s",
			la.DatabaseUser,
			la.DatabaseName,
			"disable",
			la.DatabasePort,
			la.DatabaseHost,
			la.DatabasePassword, la.DatabaseSchema))

	LOGGER.Info("Connection String ", postgresConnection.String())
	return postgresConnection.String(), nil
}

func (id *iamDB) Connect(ctx context.Context) (driver.Conn, error) {

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("Connect")
	connectionString, err := id.IAMAuth.GetConnectionString(ctx, net.LookupCNAME)
	LOGGER.Info("CONNECTION STRING : ", connectionString)
	if err != nil {
		LOGGER.Error("Could not generate connection string.")
		return nil, xerrors.Errorf("Could not get connection string: %w", err)
	}

	return GetConnectionFromDsn(ctx, connectionString)
}

func GetConnectionFromDsn(ctx context.Context, dsnString string) (driver.Conn, error) {
	LOGGER := customLogger.GetLogger()
	LOGGER.Info("GetConnectionFromDsn")
	pgxConnector := &stdlib.Driver{}
	connector, err := pgxConnector.OpenConnector(dsnString)

	if err != nil {
		LOGGER.Error("Unable to open connection with the generated connection string ", dsnString)
		return nil, xerrors.Errorf("Unable to open connection with the generated connection string: %w", err)
	}

	return connector.Connect(ctx)
}

func (ld *localDB) Connect(ctx context.Context) (driver.Conn, error) {

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("Connect")
	connectionString, err := ld.LocalAuth.GetConnectionString(ctx, net.LookupCNAME)
	if err != nil {
		LOGGER.Error("Could not generate connection string.")
		return nil, xerrors.Errorf("Could not get connection string: %w", err)
	}

	return GetConnectionFromDsn(ctx, connectionString)
}

func (id *iamDB) Driver() driver.Driver {
	return id
}

func (ld *localDB) Driver() driver.Driver {
	return ld
}

// driver.Driver interface
func (id *iamDB) Open(name string) (driver.Conn, error) {
	return nil, xerrors.New("Driver Open method is unsupported")
}

func (ld *localDB) Open(name string) (driver.Conn, error) {

	return nil, xerrors.New("Driver Open method is unsupported")
}

func (ia IAMAuth) Connect(ctx context.Context) (*sql.DB, error) {
	db := sql.OpenDB(&iamDB{ia})
	return WrapDB(db)
}

func (la LocalAuth) Connect(ctx context.Context) (*sql.DB, error) {

	db := sql.OpenDB(&localDB{la})
	return WrapDB(db)
}

func WrapDB(db *sql.DB) (*sql.DB, error) {

	LOGGER := customLogger.GetLogger()
	LOGGER.Info("WrapDB")
	db.SetConnMaxIdleTime(2 * time.Minute)
	db.SetConnMaxLifetime(13 * time.Minute)
	db.SetMaxIdleConns(2)
	db.SetMaxOpenConns(15)

	err := db.Ping()

	if err != nil {
		return nil, xerrors.Errorf("Failed to ping DB: %w", err)
	}

	LOGGER.Info("Successfully created database connection and instance. Returning the sqlx instance as wrapper for gorm.")
	return db, nil
}

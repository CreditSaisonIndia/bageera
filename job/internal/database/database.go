package database

import (
	"database/sql"
	"fmt"
	"log"
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

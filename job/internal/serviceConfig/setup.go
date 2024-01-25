package serviceConfig

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type Application struct {
	Region           string
	RunType          string
	PqJobQueueUrl    string
	EfsBasePath      string
	AlertSnsArn      string
	FileName         string
	Lpc              string
	BucketName       string
	ObjectKey        string
	InvalidObjectKey string
}

var ApplicationSetting = &Application{}

type Server struct {
	RunMode  string
	HttpPort int
}

var ServerSetting = &Server{}

type Database struct {
	Type             string
	User             string
	Password         string
	MasterDbHost     string
	ReaderDbHost     string
	Name             string
	TablePrefix      string
	Port             string
	MigrationVersion int
	SslMode          string
}

var DatabaseSetting = &Database{}

var cfg *ini.File

func SetUp(env string) {

	var err error
	var path string
	basePath := "./"

	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	// Get the directory containing the executable
	exeDir := filepath.Dir(exePath)

	// Specify the relative path to the INI file based on your directory structure
	relativePath := "internal/serviceConfig/props"
	basePath = filepath.Join(exeDir, relativePath)

	fmt.Println("Current working directory:", exePath)

	switch env {
	case "local":
		path = basePath + "/local.ini"

		cfg, err = ini.Load(path)

		if err != nil {
			log.Fatalf("setting.Setup, fail to parse %s': %v", path, err)
		}

		mapTo("application", ApplicationSetting)
		mapTo("server", ServerSetting)
		mapTo("database", DatabaseSetting)
		PrintSettings()
		break
	// Add cases for other days as needed
	default:

		ApplicationSetting.EfsBasePath = os.Getenv("efsBasePath")
		ApplicationSetting.Region = os.Getenv("region")
		ApplicationSetting.PqJobQueueUrl = os.Getenv("requestQueueUrl")
		ApplicationSetting.RunType = os.Getenv("environment")
		ApplicationSetting.AlertSnsArn = os.Getenv("alertSnsArn")
		ApplicationSetting.BucketName = os.Getenv("bucketName")
		ApplicationSetting.Lpc = os.Getenv("lpc")
		ApplicationSetting.ObjectKey = os.Getenv("objectKey")
		ApplicationSetting.FileName = os.Getenv("fileName")

		DatabaseSetting.MasterDbHost = os.Getenv("dbHost")

		DatabaseSetting.Name = "proddb"
		DatabaseSetting.Password = os.Getenv("dbPassword")
		DatabaseSetting.Port = "5432"
		DatabaseSetting.TablePrefix = "scarlet"
		DatabaseSetting.User = os.Getenv("dbUsername")
		DatabaseSetting.Type = "postgres"

		PrintSettings()

	}

}

func mapTo(section string, v interface{}) {
	err := cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}

func PrintSettings() {
	fmt.Println("Application Settings:")
	fmt.Printf("Region: %s\n", ApplicationSetting.Region)
	fmt.Printf("RunType: %s\n", ApplicationSetting.RunType)
	fmt.Printf("PqJobQueueUrl: %s\n", ApplicationSetting.PqJobQueueUrl)
	fmt.Printf("EfsBasePath: %s\n", ApplicationSetting.EfsBasePath)
	fmt.Printf("AlertSnsArn: %s\n", ApplicationSetting.AlertSnsArn)
	fmt.Printf("FileName: %s\n", ApplicationSetting.FileName)
	fmt.Printf("Lpc: %s\n", ApplicationSetting.Lpc)
	fmt.Printf("BucketName: %s\n", ApplicationSetting.BucketName)
	fmt.Printf("ObjectKey: %s\n", ApplicationSetting.ObjectKey)

	fmt.Println("\nServer Settings:")
	fmt.Printf("RunMode: %s\n", ServerSetting.RunMode)
	fmt.Printf("HttpPort: %d\n", ServerSetting.HttpPort)

	fmt.Println("\nDatabase Settings:")
	fmt.Printf("Type: %s\n", DatabaseSetting.Type)
	fmt.Printf("User: %s\n", DatabaseSetting.User)
	fmt.Printf("Password: %s\n", DatabaseSetting.Password)
	fmt.Printf("MasterDbHost: %s\n", DatabaseSetting.MasterDbHost)
	fmt.Printf("ReaderDbHost: %s\n", DatabaseSetting.ReaderDbHost)
	fmt.Printf("Name: %s\n", DatabaseSetting.Name)
	fmt.Printf("TablePrefix: %s\n", DatabaseSetting.TablePrefix)
	fmt.Printf("Port: %s\n", DatabaseSetting.Port)
	fmt.Printf("MigrationVersion: %d\n", DatabaseSetting.MigrationVersion)
}

package serviceConfig

import (
	"fmt"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"path/filepath"
)

type Application struct {
	Region        string
	RunType       string
	PqJobQueueUrl string
	EfsBasePath   string
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
	case "dev":
		path = basePath + "/dev.ini"
		break
	case "prod":
		path = basePath + "/prod.ini"
		break
	case "uat":
		path = basePath + "/uat.ini"
		break
	case "int":
		path = basePath + "/int.ini"
		break
	case "qa2":
		path = basePath + "/qa2.ini"
		break
	case "local":
		path = basePath + "/local.ini"
		break
	// Add cases for other days as needed
	default:
		path = basePath + "/local.ini"
	}

	cfg, err = ini.Load(path)

	if err != nil {
		log.Fatalf("setting.Setup, fail to parse %s': %v", path, err)
	}

	mapTo("application", ApplicationSetting)
	mapTo("server", ServerSetting)
	mapTo("database", DatabaseSetting)
}

func mapTo(section string, v interface{}) {
	err := cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}

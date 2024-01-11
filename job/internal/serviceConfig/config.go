package serviceConfig

import (
	"log"
	"os"
	"strings"
	"sync"
)

var envMap map[string]string
var envMapMutex sync.Mutex

func init() {
	// Initialize the map
	envMap = make(map[string]string)

	// Get all environment variables
	envVars := os.Environ()

	// Iterate through the environment variables, split key and value, and store them in the map
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		key := parts[0]
		value := parts[1]
		envMap[key] = value
		log.Printf("Key: %s, Value: %s\n", key, value)
	}
}

func GetEnvMap() map[string]string {
	envMapMutex.Lock()
	defer envMapMutex.Unlock()
	return envMap
}

func Get(key string) string {
	envMapMutex.Lock()
	defer envMapMutex.Unlock()
	return envMap[key]
}

// Set sets the value for a given key in the environment map
func Set(key, value string) {
	envMapMutex.Lock()
	defer envMapMutex.Unlock()
	envMap[key] = value
}

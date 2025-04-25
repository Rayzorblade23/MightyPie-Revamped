package env

import (
	"log"
	"maps"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

var envVars map[string]string

func init() {
    println("Loading environment variables...")
    var err error
    
    // Get the current file's directory
    _, filename, _, ok := runtime.Caller(0)
    if !ok {
        log.Fatal("Could not determine current file path")
    }
    
    // Construct absolute path to .env file
    envPath := filepath.Join(filepath.Dir(filename), "..","..", ".env.public")
    
    envVars, err = godotenv.Read(envPath)
    if err != nil {
        log.Printf("Error loading .env file: %v", err)
    }
}

func Get(key string) string {
    return envVars[key]
}

func GetAll() map[string]string {
    result := make(map[string]string)
    maps.Copy(result, envVars)
    return result
}
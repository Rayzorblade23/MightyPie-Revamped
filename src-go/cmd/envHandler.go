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
	envVars = make(map[string]string)

	// Get the current file's directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Could not determine current file path")
	}

	baseDir := filepath.Join(filepath.Dir(filename), "..", "..")

	// Load .env
	envPath := filepath.Join(baseDir, ".env")
	env, err := godotenv.Read(envPath)
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
	maps.Copy(envVars, env)

	// Load .env.local, overriding .env
	envLocalPath := filepath.Join(baseDir, ".env.local")
	envLocal, err := godotenv.Read(envLocalPath)
	if err != nil {
		log.Printf("Error loading .env.local file: %v", err)
	}
	maps.Copy(envVars, envLocal)
}

func Get(key string) string {
    return envVars[key]
}

func GetAll() map[string]string {
    result := make(map[string]string)
    maps.Copy(result, envVars)
    return result
}
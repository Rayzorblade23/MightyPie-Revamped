package env

import (
	"log"
	"maps"
	"path/filepath"

	"github.com/Rayzorblade23/MightyPie-Revamped/src/core"
	"github.com/joho/godotenv"
)

var envVars map[string]string

func init() {
	envVars = make(map[string]string)

	baseDir, err := core.GetRootDir()
	if err != nil {
		log.Fatalf("Could not determine base directory: %v", err)
	}

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
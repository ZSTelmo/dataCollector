package main

import (
	"datacollector/csv"
	"datacollector/database"
	"datacollector/executor"
	"datacollector/models"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Only accept workload file as command-line argument
	workloadFile := flag.String("workload", "workload.json", "Path to workload configuration file")
	flag.Parse()

	// Load workload configuration
	workload, err := models.LoadWorkloadConfig(*workloadFile)
	if err != nil {
		log.Printf("Warning: Failed to load workload file %s: %v", *workloadFile, err)
		// Initialize with default values if file cannot be loaded
		workload = &models.Workload{
			Workers:       1,
			Targets:       []string{},
			Output:        "results.csv",
			FilterPattern: "*.log",
			OutputDir:     "./output",
			OutputFile:    "query_results",
		}
	} else {
		// Ensure Workers is at least 1
		if workload.Workers <= 0 {
			log.Printf("Warning: Invalid number of workers (%d) specified in workload.json. Defaulting to 1.", workload.Workers)
			workload.Workers = 1
		}
	}

	log.Printf("Loaded workload configuration from %s: Workers=%d, Targets=%v, Output=%s, FilterPattern=%s, Query=%s",
		*workloadFile, workload.Workers, workload.Targets, workload.Output, workload.FilterPattern, workload.Query)

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	// Get database configuration from environment variables
	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "mysql" // Default to MySQL for backward compatibility
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		// Use the first target from workload.json if available, otherwise default to localhost
		if len(workload.Targets) > 0 {
			dbHost = workload.Targets[0]
			log.Printf("DB_HOST not specified in .env, using first target from workload.json: %s", dbHost)
		} else {
			dbHost = "localhost" // Default value
			log.Printf("DB_HOST not specified in .env and no targets in workload.json, using default: %s", dbHost)
		}
	}

	dbPortStr := os.Getenv("DB_PORT")
	dbPort := 3306 // Default value for MySQL
	if dbType == "postgres" && dbPortStr == "" {
		dbPort = 5432 // Default value for PostgreSQL
	} else if dbPortStr != "" {
		port, err := strconv.Atoi(dbPortStr)
		if err == nil {
			dbPort = port
		} else {
			log.Printf("Warning: Invalid DB_PORT in .env file, using default: %v", err)
		}
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root" // Default value
	}

	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSL_MODE")

	// Check required parameters
	if dbName == "" {
		log.Fatal("Database name is required. Set DB_NAME in .env file.")
	}
	if workload.Query == "" {
		log.Fatal("SQL query is required in workload configuration.")
	}
	if len(workload.Targets) == 0 {
		log.Fatal("At least one target host is required in workload configuration.")
	}

	// Log start time
	startTime := time.Now()
	log.Printf("Starting data collection at %s for targets: %v", startTime.Format(time.RFC3339), workload.Targets)

	// Create basic DB config (the host will be replaced by executor)
	dbConfig := database.Config{
		Type:     dbType,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPass,
		Database: dbName,
		SSLMode:  dbSSLMode,
	}

	// Execute queries in parallel using the executor package
	result := executor.QueryTargets(
		workload,
		dbConfig,
		dbType,
		dbPort,
		dbUser,
		dbPass,
		dbName,
		dbSSLMode,
	)

	// Check for complete failure
	if !result.HasResults && result.ErrorCount == len(workload.Targets) {
		log.Fatal("All target queries failed. No data to write.")
	}
	if !result.HasResults && result.ErrorCount < len(workload.Targets) {
		log.Printf("Warning: No data rows retrieved from any successful target.")
		// Proceed to write empty file with headers if columns were found, or just log completion
	}

	// Configure CSV output
	csvOptions := csv.WriteOptions{
		Directory:  workload.OutputDir,
		Filename:   workload.OutputFile,
		AppendDate: true,
	}

	// Write aggregated results to CSV
	if len(result.Rows) > 0 || result.HasResults { // Write even if only headers are available
		log.Printf("Aggregated %d rows from %d targets (out of %d). Writing to CSV...", 
			len(result.Rows), len(workload.Targets)-result.ErrorCount, len(workload.Targets))
		outputPath, err := csv.WriteToCSV(result.Rows, result.Columns, csvOptions)
		if err != nil {
			log.Fatalf("Failed to write aggregated data to CSV: %v", err)
		}
		// Log success
		absPath, _ := filepath.Abs(outputPath)
		log.Printf("Aggregated data successfully written to CSV file: %s", absPath)
	} else {
		log.Printf("No data rows to write to CSV.")
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	log.Printf("Process completed in %v", elapsedTime)
}

package main

import (
	"datacollector/csv"
	"datacollector/database"
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
		dbHost = "localhost" // Default value
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

	// Configure database connection
	dbConfig := database.Config{
		Type:     dbType,
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPass,
		Database: dbName,
		SSLMode:  dbSSLMode,
	}

	// Log start time
	startTime := time.Now()
	log.Printf("Starting data collection at %s", startTime.Format(time.RFC3339))
	log.Printf("Connecting to %s database %s on %s:%d", dbType, dbName, dbHost, dbPort)

	// Connect to database
	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Execute query
	log.Printf("Executing query: %s", workload.Query)
	result, err := database.ExecuteRawQuery(db, workload.Query)
	if err != nil {
		log.Fatalf("Query execution failed: %v", err)
	}

	// Close database connection when done
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Warning: Error accessing SQL DB for closing: %v", err)
	} else {
		defer sqlDB.Close()
	}

	log.Printf("Query executed successfully. Retrieved %d rows.", len(result.Rows))

	// Configure CSV output
	csvOptions := csv.WriteOptions{
		Directory:  workload.OutputDir,
		Filename:   workload.OutputFile,
		AppendDate: true,
	}

	// Write results to CSV
	outputPath, err := csv.WriteToCSV(result.Rows, result.Columns, csvOptions)
	if err != nil {
		log.Fatalf("Failed to write data to CSV: %v", err)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	// Log success
	absPath, _ := filepath.Abs(outputPath)
	log.Printf("Data successfully written to CSV file: %s", absPath)
	log.Printf("Process completed in %v", elapsedTime)
}

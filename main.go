package main

import (
	"datacollector/csv"
	"datacollector/database"
	"datacollector/models"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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

	// --- Parallel Execution Logic ---
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, workload.Workers) // Limit concurrency
	resultsChan := make(chan *database.QueryResult, len(workload.Targets))
	errChan := make(chan error, len(workload.Targets))

	for _, targetHost := range workload.Targets {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore slot

		go func(host string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot

			log.Printf("Worker starting for target: %s", host)

			// Configure database connection for this specific target
			targetDbConfig := database.Config{
				Type:     dbType,
				Host:     host, // Use the target host from the loop
				Port:     dbPort,
				User:     dbUser,
				Password: dbPass,
				Database: dbName,
				SSLMode:  dbSSLMode,
			}

			// Connect to database
			db, err := database.Connect(targetDbConfig)
			if err != nil {
				errChan <- fmt.Errorf("failed to connect to database %s on %s: %w", dbName, host, err)
				return
			}
			defer database.Close(db) // Ensure connection is closed

			// Execute query
			log.Printf("Executing query on %s: %s", host, workload.Query)
			result, err := database.ExecuteRawQuery(db, workload.Query)
			if err != nil {
				errChan <- fmt.Errorf("query execution failed on %s: %w", host, err)
				return
			}

			log.Printf("Query executed successfully on %s. Retrieved %d rows.", host, len(result.Rows))
			resultsChan <- result // Send successful result

		}(targetHost) // Pass targetHost to the goroutine
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultsChan)
	close(errChan)

	// --- Aggregation and Output ---
	var allRows [][]string
	var columns []string
	hasResults := false

	// Collect results
	for result := range resultsChan {
		if result != nil {
			if !hasResults && len(result.Columns) > 0 {
				columns = result.Columns // Get columns from the first result
				hasResults = true
			}
			if len(result.Rows) > 0 {
				allRows = append(allRows, result.Rows...)
			}
		}
	}

	// Collect and log errors
	errorCount := 0
	for err := range errChan {
		log.Printf("Error during processing: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		log.Printf("Warning: Encountered %d error(s) during parallel execution.", errorCount)
	}

	if !hasResults && errorCount == len(workload.Targets) {
		log.Fatal("All target queries failed. No data to write.")
	}
	if !hasResults && errorCount < len(workload.Targets) {
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
	if len(allRows) > 0 || hasResults { // Write even if only headers are available
		log.Printf("Aggregated %d rows from %d targets (out of %d). Writing to CSV...", len(allRows), len(workload.Targets)-errorCount, len(workload.Targets))
		outputPath, err := csv.WriteToCSV(allRows, columns, csvOptions)
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

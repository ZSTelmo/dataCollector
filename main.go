package main

import (
	"datacollector/csv"
	"datacollector/mysql"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Workload represents the configuration loaded from workload.json
type Workload struct {
	Workers       int      `json:"workers"`
	Targets       []string `json:"targets"`
	Output        string   `json:"output"`
	FilterPattern string   `json:"filter_pattern"`
}

func loadWorkloadConfig(filePath string) (*Workload, error) {
	// Read the workload.json file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Parse the JSON into the Workload struct
	var workload Workload
	if err := json.Unmarshal(data, &workload); err != nil {
		return nil, err
	}

	return &workload, nil
}

func main() {
	// Define and parse command-line flags
	// MySQL connection details come from .env file
	query := flag.String("query", "", "SQL query to execute")
	outputDir := flag.String("outdir", "./output", "Directory for output CSV files")
	outputFile := flag.String("outfile", "query_results", "Output CSV filename")
	workloadFile := flag.String("workload", "workload.json", "Path to workload configuration file")

	flag.Parse()

	// Load workload configuration
	workload, err := loadWorkloadConfig(*workloadFile)
	if err != nil {
		log.Printf("Warning: Failed to load workload file %s: %v", *workloadFile, err)
		// Initialize with default values if file cannot be loaded
		workload = &Workload{
			Workers:       1,
			Targets:       []string{},
			Output:        "results.csv",
			FilterPattern: "*.log",
		}
	}

	log.Printf("Loaded workload configuration from %s: Workers=%d, Targets=%v, Output=%s, FilterPattern=%s",
		*workloadFile, workload.Workers, workload.Targets, workload.Output, workload.FilterPattern)

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	// Get database configuration from environment variables
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost" // Default value
	}

	dbPortStr := os.Getenv("DB_PORT")
	dbPort := 3306 // Default value
	if dbPortStr != "" {
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

	// Check required parameters
	if dbName == "" {
		log.Fatal("Database name is required. Set DB_NAME in .env file.")
	}

	if *query == "" {
		log.Fatal("SQL query is required. Use -query flag.")
	}

	// Configure database connection
	dbConfig := mysql.DBConfig{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPass,
		Database: dbName,
	}

	// Log start time
	startTime := time.Now()
	log.Printf("Starting data collection at %s", startTime.Format(time.RFC3339))
	log.Printf("Connecting to MySQL database %s on %s:%d", dbName, dbHost, dbPort)

	// Connect to database
	db, err := mysql.Connect(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer mysql.Close(db)

	// Execute query
	log.Printf("Executing query: %s", *query)
	result, err := mysql.ExecuteQuery(db, *query)
	if err != nil {
		log.Fatalf("Query execution failed: %v", err)
	}

	log.Printf("Query executed successfully. Retrieved %d rows.", len(result.Rows))

	// Configure CSV output
	csvOptions := csv.WriteOptions{
		Directory:  *outputDir,
		Filename:   *outputFile,
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

// Package executor provides functionality for executing database queries in parallel
package executor

import (
	"datacollector/database"
	"datacollector/models"
	"fmt"
	"log"
	"sync"
)

// ExecutionResult represents the aggregated results of parallel query execution
type ExecutionResult struct {
	Rows       [][]string
	Columns    []string
	ErrorCount int
	HasResults bool
}

// QueryTargets executes the provided query on all target hosts in parallel
// and returns the aggregated results
func QueryTargets(
	workload *models.Workload,
	dbConfig database.Config,
	dbType string,
	dbPort int,
	dbUser string,
	dbPass string,
	dbName string,
	dbSSLMode string,
) ExecutionResult {
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

	// Return the aggregated results
	return ExecutionResult{
		Rows:       allRows,
		Columns:    columns,
		ErrorCount: errorCount,
		HasResults: hasResults,
	}
}
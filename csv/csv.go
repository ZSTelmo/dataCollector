package csv

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"datacollector/models"
)

// generateRandomString returns a random string of the specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// WriteToCSV writes the given data to a CSV file
func WriteToCSV(data [][]string, headers []string, options models.WriteOptions) (string, error) {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Create directory if it doesn't exist
	if options.Directory != "" {
		if err := os.MkdirAll(options.Directory, 0755); err != nil {
			return "", fmt.Errorf("error creating directory: %w", err)
		}
	}

	// Generate filename
	filename := options.Filename
	if options.AppendDate {
		// Add timestamp and 4 random chars to filename to make it unique
		timestamp := time.Now().Format("2006-01-02_150405")
		randomChars := generateRandomString(4)
		ext := filepath.Ext(filename)
		basename := filename[:len(filename)-len(ext)]
		filename = fmt.Sprintf("%s_%s_%s%s", basename, timestamp, randomChars, ext)
	}

	// Ensure .csv extension
	if filepath.Ext(filename) != ".csv" {
		filename = filename + ".csv"
	}

	// Create full path
	fullPath := filepath.Join(options.Directory, filename)

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("error creating CSV file: %w", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers if provided
	if len(headers) > 0 {
		if err := writer.Write(headers); err != nil {
			return "", fmt.Errorf("error writing headers to CSV: %w", err)
		}
	}

	// Write data rows
	if err := writer.WriteAll(data); err != nil {
		return "", fmt.Errorf("error writing data to CSV: %w", err)
	}

	return fullPath, nil
}

// AppendToCSV appends data to an existing CSV file or creates a new one if it doesn't exist
func AppendToCSV(data [][]string, filePath string, writeHeaders bool, headers []string) error {
	// Check if file exists to determine if we need to write headers
	fileExists := false
	if _, err := os.Stat(filePath); err == nil {
		fileExists = true
	}

	// Open file in append mode or create it
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening/creating CSV file: %w", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers if the file is new and headers are provided
	if !fileExists && writeHeaders && len(headers) > 0 {
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("error writing headers to CSV: %w", err)
		}
	}

	// Write data rows
	if err := writer.WriteAll(data); err != nil {
		return fmt.Errorf("error writing data to CSV: %w", err)
	}

	return nil
}

// ReadCSV reads data from a CSV file
func ReadCSV(filePath string) ([][]string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file: %w", err)
	}

	return records, nil
}

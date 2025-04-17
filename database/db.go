package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds database configuration
type Config struct {
	Type     string // "mysql" or "postgres"
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string // For PostgreSQL
}

// QueryResult represents a query result set
type QueryResult struct {
	Columns []string
	Rows    [][]string
}

// Connect establishes a connection to the database using GORM
func Connect(config Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Configure database connection based on type
	switch config.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.User, config.Password, config.Host, config.Port, config.Database)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		})

	case "postgres":
		sslMode := config.SSLMode
		if sslMode == "" {
			sslMode = "disable" // Default SSL mode
		}
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=UTC",
			config.Host, config.User, config.Password, config.Database, config.Port, sslMode)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: gormLogger,
		})

	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported types: mysql, postgres)", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error accessing underlying SQL DB: %w", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Minute * 3)

	// Check if connection is working
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	return db, nil
}

// ExecuteRawQuery executes the given SQL query and returns the result
func ExecuteRawQuery(db *gorm.DB, query string) (*QueryResult, error) {
	// Execute raw query
	rows, err := db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting column names: %w", err)
	}

	// Create result set
	result := &QueryResult{
		Columns: columns,
		Rows:    [][]string{},
	}

	// Prepare containers for row data
	columnCount := len(columns)
	values := make([]interface{}, columnCount)
	valuePtrs := make([]interface{}, columnCount)

	// Fetch rows
	for rows.Next() {
		// Initialize with new values for each row
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		// Convert to strings
		rowStrings := make([]string, columnCount)
		for i, val := range values {
			if val == nil {
				rowStrings[i] = "NULL"
			} else {
				// Handle different types of values
				switch v := val.(type) {
				case []byte:
					rowStrings[i] = string(v)
				default:
					rowStrings[i] = fmt.Sprintf("%v", v)
				}
			}
		}

		result.Rows = append(result.Rows, rowStrings)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading rows: %w", err)
	}

	return result, nil
}

// Close safely closes the database connection
func Close(db *gorm.DB) error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("error accessing SQL DB: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}

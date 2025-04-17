# Data Collector

A Go application that collects data from MySQL and PostgreSQL databases and exports it to CSV files for further analysis.

## Overview

This tool executes SQL queries against databases (supporting both MySQL and PostgreSQL) and writes the results to CSV files with customizable options. It supports configuration through both command-line arguments and a workload configuration file.

## Features

- Connect to MySQL or PostgreSQL databases using configurable connection parameters
- Execute custom SQL queries
- Export query results to CSV files
- Customize output directory and filenames
- Automatically append timestamps to filenames
- Configure multiple parameters through a workload.json file
- Environment variable support through .env files

## Requirements

- Go 1.16 or higher
- MySQL or PostgreSQL database server
- Required Go packages:
  - github.com/joho/godotenv
  - gorm.io/gorm
  - gorm.io/driver/mysql
  - gorm.io/driver/postgres

## Installation

1. Clone the repository:
   ```
   git clone <repository-url>
   cd dataCollector
   ```

2. Install dependencies:
   ```
   go mod download
   ```

## Configuration

### Environment Variables

Create a `.env` file in the project root with the following variables:

```
DB_TYPE=mysql           # Options: 'mysql' or 'postgres'
DB_HOST=localhost
DB_PORT=3306            # Default: 3306 for MySQL, 5432 for PostgreSQL
DB_USER=root
DB_PASSWORD=yourpassword
DB_NAME=yourdatabase
DB_SSL_MODE=disable     # For PostgreSQL: disable, require, verify-ca, verify-full
```

### Workload Configuration

You can customize the execution using a `workload.json` file:

```json
{
  "workers": 4,
  "targets": [],
  "output": "results.csv",
  "filter_pattern": "*.log"
}
```

- `workers`: Number of concurrent workers (integer)
- `targets`: List of target sources (array of strings)
- `output`: Default output filename (string)
- `filter_pattern`: File pattern for filtering (string)

## Usage

Basic usage:

```
go run main.go -query "SELECT * FROM your_table"
```

With additional options:

```
go run main.go -query "SELECT * FROM your_table" -outdir "./data" -outfile "export" -workload "custom-workload.json"
```

### Command-line Arguments

- `-query`: SQL query to execute (required)
- `-outdir`: Directory for output CSV files (default: "./output")
- `-outfile`: Output CSV filename without extension (default: "query_results")
- `-workload`: Path to workload configuration file (default: "workload.json")

## Output

The application produces CSV files with:
- One row for column headers
- Result data from the SQL query
- Timestamps in filenames by default (e.g., `query_results_2025-04-16_120000.csv`)

## Example

1. Set up your database connection in `.env` file (MySQL example):
   ```
   DB_TYPE=mysql
   DB_HOST=localhost
   DB_PORT=3306
   DB_USER=root
   DB_PASSWORD=yourpassword
   DB_NAME=yourdatabase
   ```

2. Or for PostgreSQL:
   ```
   DB_TYPE=postgres
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=yourpassword
   DB_NAME=yourdatabase
   DB_SSL_MODE=disable
   ```

3. Run a query:
   ```
   go run main.go -query "SELECT id, name, email FROM customers WHERE created_at > '2025-01-01'"
   ```
   
4. Check the output directory for your CSV file with the query results

## Project Structure

- `main.go`: Main application logic and command parsing
- `database/db.go`: Database connection and query execution with ORM support
- `csv/csv.go`: CSV file writing and manipulation
- `workload.json`: Default workload configuration

## Error Handling

The application provides detailed error messages and warnings:
- Database connection issues
- Query execution failures 
- File and directory access problems
- Configuration loading errors

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here]
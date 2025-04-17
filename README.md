# Data Collector

A Go application that collects data by executing SQL queries against multiple MySQL or PostgreSQL databases in parallel and aggregates the results into a single CSV file.

## Overview

This tool executes a specified SQL query against a list of target databases (supporting both MySQL and PostgreSQL) concurrently. It aggregates the results from all successful queries and writes them to a single CSV file with customizable options. Configuration is primarily managed through a `workload.json` file and environment variables (`.env`).

## Features

- Connect to multiple MySQL or PostgreSQL databases
- Execute a custom SQL query concurrently across specified target databases
- Aggregate results from multiple databases into a single CSV file
- Limit concurrency using a configurable number of workers
- Customize output directory and filenames
- Automatically append timestamps to filenames
- Configure database connections and execution parameters through `workload.json` and `.env` files

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

Configuration is managed through environment variables (`.env` file) and a workload configuration file (`workload.json`).

### Environment Variables

Create a `.env` file in the project root with the following variables for database connection details. These settings apply to all target databases unless overridden by specific target configurations (if implemented in the future).

```
DB_TYPE=mysql           # Options: 'mysql' or 'postgres'
DB_HOST=fallback_host   # Fallback host if 'targets' in workload.json is empty (optional)
DB_PORT=3306            # Default: 3306 for MySQL, 5432 for PostgreSQL
DB_USER=root
DB_PASSWORD=yourpassword
DB_NAME=yourdatabase    # Database name (required)
DB_SSL_MODE=disable     # For PostgreSQL: disable, require, verify-ca, verify-full
```
**Note:** The primary list of database hosts to query is defined in `workload.json`. `DB_HOST` in `.env` is only used as a fallback if the `targets` list in `workload.json` is empty.

### Workload Configuration

Customize the execution behavior using a `workload.json` file (or specify a different file using the `-workload` flag).

```json
{
  "workers": 4,
  "targets": ["db1.example.com", "db2.example.com", "192.168.1.100"],
  "query": "SELECT id, name, status FROM tasks WHERE status = 'pending'",
  "output_dir": "./output",
  "output_file": "query_results",
  "filter_pattern": "*.log" // Note: filter_pattern seems unused in the current main.go logic
}
```

- `workers`: (Integer) Maximum number of concurrent database query executions. Defaults to 1 if not specified or invalid.
- `targets`: (Array of strings, Required) List of database hostnames or IP addresses to query. At least one target is required.
- `query`: (String, Required) The SQL query to execute on each target database.
- `output_dir`: (String) Directory where the output CSV file will be saved (default: "./output").
- `output_file`: (String) Base filename for the output CSV file (default: "query_results"). A timestamp will be appended.
- `filter_pattern`: (String) Currently unused in the main data collection logic.

## Usage

Run the application from the command line. Configuration is primarily done via `.env` and `workload.json`.

```bash
go run main.go
```

To use a specific workload file:

```bash
go run main.go -workload "path/to/your/custom-workload.json"
```

### Command-line Arguments

- `-workload`: Path to the workload configuration JSON file (default: "workload.json").

## Output

The application produces a single CSV file in the specified `output_dir`.
- The filename is based on `output_file` with an appended timestamp (e.g., `query_results_2025-04-17_103000.csv`).
- The file contains aggregated results from all target databases where the query executed successfully.
- The first row contains the column headers from the query.
- Subsequent rows contain the data retrieved from the databases.

## Example

1.  **Configure `.env`:**
    Set up your default database connection details (user, password, db name, type). `DB_HOST` is optional if `targets` is set in `workload.json`.
    ```
    DB_TYPE=mysql
    DB_USER=app_user
    DB_PASSWORD=secret
    DB_NAME=inventory
    ```

2.  **Configure `workload.json`:**
    Define the target databases, the query, and worker count.
    ```json
    {
      "workers": 5,
      "targets": ["prod-db-1.region1.local", "prod-db-2.region1.local", "prod-db-1.region2.local"],
      "query": "SELECT hostname, cpu_usage, memory_usage FROM server_metrics WHERE cpu_usage > 90.0",
      "output_dir": "./results/high_cpu",
      "output_file": "high_cpu_servers"
    }
    ```

3.  **Run the application:**
    ```bash
    go run main.go
    ```

4.  **Check the output:**
    Look for a CSV file like `results/high_cpu/high_cpu_servers_YYYY-MM-DD_HHMMSS.csv` containing aggregated data from the specified targets.

## Project Structure

- `main.go`: Main application logic, configuration loading, parallel execution orchestration.
- `database/db.go`: Database connection and query execution with ORM support
- `csv/csv.go`: CSV file writing and manipulation
- `workload.json`: Default workload configuration

## Error Handling

The application provides detailed logging for:
- Configuration loading issues (`.env`, `workload.json`)
- Database connection failures (per target)
- Query execution failures (per target)
- CSV file writing problems

Errors encountered during connection or query execution for individual targets are logged, but the application attempts to continue processing other targets. It will only exit fatally if essential configuration is missing or if *all* target queries fail. A summary of errors encountered is logged at the end of the process.

## License

[Add your license information here]

## Contributing

[Add contribution guidelines here]
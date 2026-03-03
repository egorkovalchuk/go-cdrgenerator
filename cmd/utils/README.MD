# LAC/CELL Pool Generation Utility
This project is a utility for generating a pool of LAC (Location Area Code) and CELL (cell identifier) data based on configuration settings and queries to an Oracle database.

## Description
The utility performs the following tasks:
* Reading configuration from a JSON file.
* Connecting to the Oracle database.
* Executing SQL queries to retrieve LAC and CELL data.
* Saving the results to a CSV file.

## Usage
Command-line Flags
-pool: Launches the utility to create a LAC/CELL pool.
-t: Task name (mandatory parameter).
-p: Password for connecting to the database (mandatory parameter).
-m: Runs all tasks specified in the configuration.

## Example
```bash
go run main.go -pool -t task_name -p password
```
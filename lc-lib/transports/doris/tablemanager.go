/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package doris

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-lib/addresspool"
)

// tableManager handles Doris table creation and column management
type tableManager struct {
	config     *TransportDorisFactory
	table      string
	columnDefs map[string]string
	indexDefs  []string
}

// newTableManager creates a new table manager instance
func newTableManager(config *TransportDorisFactory, table string) *tableManager {
	return &tableManager{
		config: config,
		table:  table,
	}
}

// InitializeSchema initializes column definitions and ensures the table exists with necessary columns
// Returns (connected, error) where connected indicates if SQL connection was successful
func (tm *tableManager) InitializeSchema(poolEntry *addresspool.PoolEntry, addr *addresspool.Address) (bool, error) {
	// Initialize column definitions with hard-coded defaults
	tm.columnDefs = map[string]string{
		"@timestamp":             "datetime",
		"type":                   "varchar(255)",
		"host":                   "varchar(255)",
		"path":                   "varchar(5120)",
		"offset":                 "bigint",
		"tags":                   "array<varchar(255)>",
		"message":                "text",
		tm.config.RestJSONColumn: "variant",
	}

	tm.indexDefs = []string{
		"INDEX `idx_tags`(`tags`) USING INVERTED",
		"INDEX `idx_message`(`message`) USING INVERTED PROPERTIES(\"parser\"=\"basic\")",
		fmt.Sprintf("INDEX `idx_%[1]s`(`%[1]s`) USING INVERTED PROPERTIES(\"parser\"=\"basic\")", tm.config.RestJSONColumn),
	}

	// Add additional columns from configuration
	for colName, colType := range tm.config.additionalColumnDefs {
		tm.columnDefs[colName] = colType
	}

	// Establish SQL connection
	db, err := tm.connectSQL(poolEntry, addr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	// Check if table exists using DESCRIBE
	describeSQL := fmt.Sprintf("DESCRIBE `%s`.`%s`", tm.config.Database, tm.table)
	rows, err := db.Query(describeSQL)
	if err != nil {
		// Table likely doesn't exist - create it
		return true, tm.createTable(poolEntry, addr, db)
	}
	defer rows.Close()

	// Table exists - check columns
	return true, tm.validateAndUpdateColumns(poolEntry, addr, db, rows)
}

// ColumnDefs returns the column definitions
func (tm *tableManager) ColumnDefs() map[string]string {
	return tm.columnDefs
}

// connectSQL establishes a SQL connection to Doris
func (tm *tableManager) connectSQL(poolEntry *addresspool.PoolEntry, addr *addresspool.Address) (*sql.DB, error) {
	var dsn string
	log.Infof("[T %s]{%s} Connecting to Doris with username: %s", poolEntry.Server, addr.Desc(), tm.config.Username)
	if tm.config.Username != "" && tm.config.Password != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s", tm.config.Username, tm.config.Password, addr.Addr().String(), tm.config.Database)
	} else {
		dsn = fmt.Sprintf("@tcp(%s)/%s", addr.Addr().String(), tm.config.Database)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQL connection: %s", err)
	}

	// Test the connection with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQL connection: %s", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute * 5)

	log.Infof("[T %s]{%s} Established SQL connection to Doris", poolEntry.Server, addr.Desc())
	return db, nil
}

// validateAndUpdateColumns validates existing columns and adds missing ones
func (tm *tableManager) validateAndUpdateColumns(poolEntry *addresspool.PoolEntry, addr *addresspool.Address, db *sql.DB, rows *sql.Rows) error {
	log.Infof("[T %s]{%s} Validating existing table schema for %s.%s", poolEntry.Server, addr.Desc(), tm.config.Database, tm.table)
	// Parse DESCRIBE result to get existing columns
	existingCols := make(map[string]string)

	// Get column names from the result set
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column names: %s", err)
	}

	// Create a slice to hold the row values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan the rows
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %s", err)
		}

		// First column is Field (column name), second is Type
		if len(values) >= 2 {
			colName := ""
			colType := ""

			if v, ok := values[0].([]byte); ok {
				colName = string(v)
			}
			if v, ok := values[1].([]byte); ok {
				colType = string(v)
			}

			if colName != "" && colType != "" {
				existingCols[colName] = colType
			}
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading rows: %s", err)
	}

	// Check for columns with wrong types
	for colName, expectedType := range tm.columnDefs {
		if existingType, exists := existingCols[colName]; exists {
			// Normalize type comparison
			if !strings.EqualFold(normalizeType(existingType), normalizeType(expectedType)) {
				return fmt.Errorf("column '%s' has type '%s' but expected '%s' - manual schema fix needed", colName, existingType, expectedType)
			}
		}
	}

	// Add missing columns
	var missingCols []string
	for colName := range tm.columnDefs {
		if _, exists := existingCols[colName]; !exists {
			missingCols = append(missingCols, colName)
		}
	}

	if len(missingCols) == 0 {
		log.Infof("[T %s]{%s} Doris table %s.%s schema is valid", poolEntry.Server, addr.Desc(), tm.config.Database, tm.table)
		return nil
	}

	// Add missing columns
	for _, colName := range missingCols {
		colType := tm.columnDefs[colName]
		alterSQL := fmt.Sprintf("ALTER TABLE `%s`.`%s` ADD COLUMN `%s` %s", tm.config.Database, tm.table, colName, colType)

		_, err := db.Exec(alterSQL)
		if err != nil {
			return fmt.Errorf("failed to add column '%s': %s", colName, err)
		}

		log.Infof("[T %s]{%s} Added column '%s' to table %s.%s", poolEntry.Server, addr.Desc(), colName, tm.config.Database, tm.table)
	}

	return nil
}

// createTable creates a new table with proper schema and partitioning
func (tm *tableManager) createTable(poolEntry *addresspool.PoolEntry, addr *addresspool.Address, db *sql.DB) error {
	log.Infof("[T %s]{%s} Creating new table %s.%s", poolEntry.Server, addr.Desc(), tm.config.Database, tm.table)

	columnDefs := make([]string, 0, len(tm.columnDefs))
	for colName, colType := range tm.columnDefs {
		columnDefs = append(columnDefs, fmt.Sprintf("`%s` %s", colName, colType))
	}
	// Sort by column name for consistency but force @timestamp first, and type second
	sort.SliceStable(columnDefs, func(i, j int) bool {
		if strings.HasPrefix(columnDefs[i], "`@timestamp`") {
			return true
		}
		if strings.HasPrefix(columnDefs[j], "`@timestamp`") {
			return false
		}
		if strings.HasPrefix(columnDefs[i], "`type`") {
			return true
		}
		if strings.HasPrefix(columnDefs[j], "`type`") {
			return false
		}
		return columnDefs[i] < columnDefs[j]
	})

	// Build properties including replication and partition retention
	properties := []string{
		`"replication_num" = "1"`,
		`"dynamic_partition.enable" = "true"`,
		`"dynamic_partition.time_unit" = "DAY"`,
		fmt.Sprintf(`"dynamic_partition.start" = "-%d"`, tm.config.PartitionRetentionDays),
		`"dynamic_partition.end" = "3"`,
		`"dynamic_partition.prefix" = "p"`,
		`"dynamic_partition.buckets" = "10"`,
	}

	createSQL := fmt.Sprintf(
		"CREATE TABLE `%s`.`%s` "+
			"(%s, %s) "+
			"DUPLICATE KEY(`@timestamp`, `type`) "+
			"PARTITION BY RANGE(`@timestamp`) () "+
			"DISTRIBUTED BY HASH(`type`) BUCKETS 10 "+
			"PROPERTIES (%s)",
		tm.config.Database,
		tm.table,
		strings.Join(columnDefs, ", "),
		strings.Join(tm.indexDefs, ", "),
		strings.Join(properties, ", "),
	)

	_, err := db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %s", err)
	}

	log.Infof("[T %s]{%s} Created Doris table %s.%s with %d-day retention", poolEntry.Server, addr.Desc(), tm.config.Database, tm.table, tm.config.PartitionRetentionDays)
	return nil
}

// normalizeType normalizes a Doris type for comparison
func normalizeType(t string) string {
	return strings.ToUpper(strings.TrimSpace(t))
}

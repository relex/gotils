package mssqlutil

import (
	"database/sql"
	"fmt"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/relex/gotils/logger"
)

// BulkInsert performs SQL Server bulk-insert from input rows represented by (rowCount, getRow)
//
// No reflection here. The getRow parameter must transform source data fields into formats compatible to the destination columns
func BulkInsert(tx *sql.Tx, tableName string, columnNames []string, rowCount int, getRow func(index int) []interface{}) (int64, error) {
	stmt, stmtErr := tx.Prepare(mssql.CopyIn(tableName, mssql.BulkOptions{}, columnNames...))
	if stmtErr != nil {
		return 0, fmt.Errorf("failed to prepare bulk insert statement: %w", stmtErr)
	}

	for i := 0; i < rowCount; i++ {
		row := getRow(i)
		if len(row) != len(columnNames) {
			logger.WithField("table", tableName).Panicf("bulkInsert: wrong numbers of values in row #d: %v", row)
		}

		_, appendErr := stmt.Exec(row...)
		if appendErr != nil {
			return 0, fmt.Errorf("failed to append locally: row #%d %v: %w", i, row, appendErr)
		}
	}

	result, execErr := stmt.Exec()
	if execErr != nil {
		return 0, fmt.Errorf("failed to execute bulk insert: %w", execErr)
	}

	count, countErr := result.RowsAffected()
	if countErr != nil {
		return 0, fmt.Errorf("failed to count inserted rows: %w", countErr)
	}

	if err := stmt.Close(); err != nil {
		return count, fmt.Errorf("failed to close bulk insert statement: %w", err)
	}

	return count, nil
}

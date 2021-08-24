package dbutil

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/relex/gotils/logger"
)

const azureSqlRetryAttempts = 10

func RunSession(sqlURL string, do func(tx *sql.Tx) error) {
	var retryAttempts int
	if strings.Contains(sqlURL, "database.windows.net") {
		retryAttempts = azureSqlRetryAttempts
	} else {
		retryAttempts = 0
	}

	db, dbErr := sql.Open("sqlserver", sqlURL)
	if dbErr != nil {
		logger.Fatalf("failed to open DB driver: %v", dbErr)
	}
	defer db.Close()

	var round = 0
	var conn *sql.Conn
	var connErr error
	for {
		round++
		conn, connErr = db.Conn(context.Background())
		if connErr != nil {
			if round > retryAttempts || !strings.Contains(connErr.Error(), " is not currently available") {
				logger.Fatalf("failed to connect to DB: %v", connErr)
			}
		} else {
			break
		}
		logger.Warnf("reconnect attempt #%d after %v", round, connErr)
	}
	defer conn.Close()

	tx, txErr := conn.BeginTx(context.Background(), nil)
	if txErr != nil {
		logger.Fatalf("failed to begin transaction: %v", txErr)
	}

	if err := do(tx); err != nil {
		logger.Fatalf("failed during DB session: %v", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Fatalf("failed to commit ")
	}
}

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

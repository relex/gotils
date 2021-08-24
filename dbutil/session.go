package dbutil

import (
	"context"
	"database/sql"
	"strings"

	"github.com/relex/gotils/logger"
)

const azureSqlRetryAttempts = 10

// RunSession runs a simple DB session with all actions enclosed within a transaction
//
// It connects to DB, starts a transaction, calls "do" and then commits it.
//
// Special handling for Azure SQL Server, which are often unavailable temporarily
func RunSession(driver string, url string, do func(tx *sql.Tx) error) {
	var retryAttempts int
	if strings.Contains(url, "database.windows.net") {
		retryAttempts = azureSqlRetryAttempts
	} else {
		retryAttempts = 0
	}

	db, dbErr := sql.Open(driver, url)
	if dbErr != nil {
		logger.Fatalf("failed to open DB driver '%s': %v", driver, dbErr)
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

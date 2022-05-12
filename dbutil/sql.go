package dbutil

import (
	"database/sql"
	"fmt"
)

// ExecOne executes a query within the given transaction and returns the number of affected rows
func ExecOne(tx *sql.Tx, query string, args ...interface{}) (int64, error) {
	result, execErr := tx.Exec(query, args...)
	if execErr != nil {
		return 0, fmt.Errorf("failed to execute: %w", execErr)
	}

	count, countErr := result.RowsAffected()
	if countErr != nil {
		return 0, fmt.Errorf("failed to count affected rows: %w", countErr)
	}

	return count, countErr
}

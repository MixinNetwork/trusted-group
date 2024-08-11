package mtg

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (s *SQLite3Store) ReadCache(ctx context.Context, k string) (string, error) {
	row := s.db.QueryRowContext(ctx, "SELECT value FROM caches WHERE key=?", k)
	var value string
	err := row.Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SQLite3Store) WriteCache(ctx context.Context, k, v string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	threshold := time.Now().Add(-24 * time.Hour).UTC()
	_, err = tx.ExecContext(ctx, "DELETE FROM caches WHERE created_at<?", threshold)
	if err != nil {
		return err
	}

	cols := []string{"key", "value", "created_at"}
	vals := []any{k, v, time.Now().UTC()}
	err = s.execOne(ctx, tx, buildInsertionSQL("caches", cols), vals...)
	if err != nil {
		return fmt.Errorf("INSERT caches %v", err)
	}
	return tx.Commit()
}

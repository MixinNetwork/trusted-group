package mtg

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const cacheTTL = 24 * time.Hour

func (s *SQLite3Store) ReadCache(ctx context.Context, k string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	row := s.db.QueryRowContext(ctx, "SELECT value,created_at FROM caches WHERE key=?", k)
	var value string
	var createdAt time.Time
	err := row.Scan(&value, &createdAt)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", err
	}
	if createdAt.Add(cacheTTL).Before(time.Now()) {
		return "", nil
	}
	return value, nil
}

func (s *SQLite3Store) WriteCache(ctx context.Context, k, v string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollBack(tx)

	threshold := time.Now().Add(-cacheTTL).UTC()
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

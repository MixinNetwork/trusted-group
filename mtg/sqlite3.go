package mtg

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var SCHEMA string

type SQLite3Store struct {
	db    *sql.DB
	mutex *sync.RWMutex
}

func ExpandTilde(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	path = strings.Replace(path, "~", home, 1)
	return path
}

func OpenSQLite3Store(path string) (*SQLite3Store, error) {
	path = ExpandTilde(path)
	dsn := fmt.Sprintf("file:%s?mode=rwc&_journal_mode=WAL&cache=private", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(SCHEMA)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return &SQLite3Store{
		db:    db,
		mutex: new(sync.RWMutex),
	}, nil
}

func (s *SQLite3Store) Close() error {
	return s.db.Close()
}

func (s *SQLite3Store) execOne(ctx context.Context, tx *sql.Tx, sql string, params ...any) error {
	return s.execMultiple(ctx, tx, 1, sql, params...)
}

func (s *SQLite3Store) execMultiple(ctx context.Context, tx *sql.Tx, num int64, sql string, params ...any) error {
	res, err := tx.ExecContext(ctx, sql, params...)
	logger.Verbosef("SQLite3Store.ExecContext(%s, %v) => %v", sql, params, err)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows != num {
		return fmt.Errorf("exec(%d, %s) => %d %v", num, sql, rows, err)
	}
	return nil
}

func buildInsertionSQL(table string, cols []string) string {
	vals := strings.Repeat("?, ", len(cols))
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ","), vals[:len(vals)-2])
}

func (s *SQLite3Store) checkExistence(ctx context.Context, tx *sql.Tx, sql string, params ...any) (bool, error) {
	rows, err := tx.QueryContext(ctx, sql, params...)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (s *SQLite3Store) ReadProperty(ctx context.Context, k string) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	row := s.db.QueryRowContext(ctx, "SELECT value FROM properties WHERE key=?", k)
	var value string
	err := row.Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SQLite3Store) WriteProperty(ctx context.Context, k, v string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollBack(tx)

	existed, err := s.checkExistence(ctx, tx, "SELECT value FROM properties WHERE key=?", k)
	if err != nil {
		return err
	}

	createdAt := time.Now().UTC()
	if existed {
		err = s.execOne(ctx, tx, "UPDATE properties SET value=?, updated_at=? WHERE key=?", v, createdAt, k)
		if err != nil {
			return fmt.Errorf("UPDATE properties %v", err)
		}
	} else {
		cols := []string{"key", "value", "created_at", "updated_at"}
		err = s.execOne(ctx, tx, buildInsertionSQL("properties", cols), k, v, createdAt, createdAt)
		if err != nil {
			return fmt.Errorf("INSERT properties %v", err)
		}
	}
	return tx.Commit()
}

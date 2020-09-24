package durable

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"multisig/configs"
	"time"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type Database struct {
	db *sqlx.DB
}

func OpenDatabaseClient(c *configs.Option) *Database {
	database := c.Database
	conn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", database.User, database.Password, database.Host, database.Port, database.Name)
	db, err := sqlx.Connect("pgx", conn)
	if err != nil {
		log.Panicln(err)
	}
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxOpenConns(128)
	db.SetMaxIdleConns(4)

	return &Database{db: db}
}

func (d *Database) RunInTransaction(ctx context.Context, opts *sql.TxOptions, fn func(*sqlx.Tx) error) error {
	tx, err := d.db.BeginTxx(ctx, opts)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	return tx.Commit()
}

func (d *Database) MustExec(query string, args ...interface{}) sql.Result {
	return d.db.MustExec(query, args...)
}

func (d *Database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *Database) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return d.db.SelectContext(ctx, dest, query, args...)
}

func (d *Database) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return d.db.GetContext(ctx, dest, query, args...)
}

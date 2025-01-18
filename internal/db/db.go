package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
)

const PGMigrantSchema = "pgmigrant"
const MigrationTableName = PGMigrantSchema + ".current_version"

var ErrTableNotFound = errors.New("table not found")

type Conn struct {
	*sql.DB
}

func NewConnEnsureVersionTable(ctx context.Context, url string) (*Conn, string, error) {
	conn, err := NewConn(ctx, url)
	if err != nil {
		return nil, "", err
	}
	currentVersion, err := conn.CheckCurrentVersion(ctx)
	if err != nil {
		if errors.Is(err, ErrTableNotFound) {
			if err = conn.CreateMigrationTable(ctx); err != nil {
				return nil, "", err
			}
		} else {
			return nil, "", err
		}
	}
	return conn, currentVersion, nil
}

func NewConn(ctx context.Context, url string) (*Conn, error) {
	connConfig, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	conn := stdlib.OpenDB(*connConfig)
	return &Conn{conn}, nil
}

func (c *Conn) Close(ctx context.Context) error {
	return c.DB.Close()
}

func (c *Conn) CreateMigrationTable(ctx context.Context) error {
	if _, err := c.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+PGMigrantSchema+`;`); err != nil {
		fmt.Printf("error creating schema: %v", err)
		return err
	}
	if _, err := c.ExecContext(ctx, `
		CREATE TABLE `+MigrationTableName+` (
			id INTEGER PRIMARY KEY CHECK (id = 1) DEFAULT 1,
			version text NOT NULL,
			created_at timestamptz NOT NULL DEFAULT now()
		);
	`); err != nil {
		return err
	}
	return nil
}

func (c *Conn) CheckCurrentVersion(ctx context.Context) (string, error) {
	var version string
	err := c.QueryRowContext(ctx, `SELECT version FROM `+MigrationTableName).Scan(&version)
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "42P01":
			return "", ErrTableNotFound
		case err == sql.ErrNoRows:
			return "", nil
		default:
			return "", err
		}
	}
	return version, nil
}

var (
	defaultTimeout     = 90 * time.Second
	defaultLockTimeout = 60 * time.Second
)

func (c *Conn) ApplyMigration(ctx context.Context, version, sql string) error {
	start := time.Now()
	tx, err := c.BeginTx(ctx, nil)
	defer tx.Rollback() // No-op if committed successfully
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	// Due to the way *sql.Db works, when a statement_timeout is set for the session, it will NOT reset
	// by default when it's returned to the pool.
	//
	// We can't set the timeout at the TRANSACTION-level (for each transaction) because `ADD INDEX CONCURRENTLY`
	// must be executed within its own transaction block. Postgres will error if you try to set a TRANSACTION-level
	// timeout for it. SESSION-level statement_timeouts are respected by `ADD INDEX CONCURRENTLY`
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET SESSION statement_timeout = %d", defaultTimeout.Milliseconds())); err != nil {
		return fmt.Errorf("setting statement timeout: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET SESSION lock_timeout = %d", defaultLockTimeout.Milliseconds())); err != nil {
		return fmt.Errorf("setting lock timeout: %w", err)
	}
	if _, err := tx.ExecContext(ctx, sql); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO `+MigrationTableName+` (id, version) VALUES (1, $1)
		ON CONFLICT (id) DO UPDATE SET version = $1;`, version); err != nil {
		return fmt.Errorf("failed to update current version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	fmt.Printf("\n✅ Finished executing statement. Duration: %s\n", time.Since(start))
	return nil
}

func (c *Conn) CleanSchema(ctx context.Context) error {
	if _, err := c.ExecContext(ctx, `DROP SCHEMA IF EXISTS `+PGMigrantSchema+` CASCADE;`); err != nil {
		fmt.Printf("error cleaning pg-migrant schema: %v", err)
		return err
	}
	if _, err := c.ExecContext(ctx, `DROP SCHEMA IF EXISTS public CASCADE;`); err != nil {
		fmt.Printf("error cleaning public schema: %v", err)
		return err
	}
	if _, err := c.ExecContext(ctx, `CREATE SCHEMA public;`); err != nil {
		fmt.Printf("error creating public schema: %v", err)
		return err
	}
	return nil
}

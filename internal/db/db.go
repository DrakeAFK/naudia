package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	SQL             *sql.DB
	Path            string
	VectorAvailable bool
	logger          *log.Logger
}

func Open(ctx context.Context, path string, logger *log.Logger) (*DB, error) {
	if err := ensureParent(path); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	store := &DB{SQL: conn, Path: path, logger: logger}
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := store.Migrate(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return store, nil
}

func (d *DB) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}
	return d.SQL.Close()
}

func (d *DB) Migrate(ctx context.Context) error {
	if _, err := d.SQL.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL);`); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		applied, err := d.migrationApplied(ctx, version)
		if err != nil {
			return err
		}
		if applied {
			if strings.HasPrefix(version, "002_") {
				d.VectorAvailable = d.vectorTableExists(ctx)
			}
			continue
		}
		sqlText, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if strings.HasPrefix(version, "002_") {
			if _, err := d.SQL.ExecContext(ctx, string(sqlText)); err != nil {
				d.VectorAvailable = false
				if d.logger != nil {
					d.logger.Printf("sqlite-vec unavailable: %v", err)
				}
				continue
			}
			d.VectorAvailable = true
		} else {
			if _, err := d.SQL.ExecContext(ctx, string(sqlText)); err != nil {
				return fmt.Errorf("migration %s failed: %w", version, err)
			}
		}
		if _, err := d.SQL.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?, datetime('now'))`, version); err != nil {
			return err
		}
	}
	d.VectorAvailable = d.vectorTableExists(ctx)
	return nil
}

func (d *DB) migrationApplied(ctx context.Context, version string) (bool, error) {
	var count int
	err := d.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&count)
	return count > 0, err
}

func (d *DB) vectorTableExists(ctx context.Context) bool {
	var name string
	err := d.SQL.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'vec_chunks'`).Scan(&name)
	if err == nil && name == "vec_chunks" {
		return true
	}
	err = d.SQL.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'vec_chunks_shadow'`).Scan(&name)
	return err == nil
}

func (d *DB) Tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func ensureParent(path string) error {
	if path == "" {
		return errors.New("database path is empty")
	}
	return mkdirAll(filepath.Dir(path))
}

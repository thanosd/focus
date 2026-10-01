package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Migrator struct {
	db            *sql.DB
	migrationsDir string
}

func NewMigrator(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

func (m *Migrator) Migrate() error {
	// 1. Ensure migrations table exists
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// 2. Get applied migrations
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// 3. Read migration files
	files, err := os.ReadDir(m.migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort files
	var migrationFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			migrationFiles = append(migrationFiles, f.Name())
		}
	}
	sort.Strings(migrationFiles)

	// 4. Apply new migrations
	for _, file := range migrationFiles {
		if _, ok := applied[file]; ok {
			continue
		}

		fmt.Printf("Applying migration: %s... ", file)
		if err := m.applyMigration(file); err != nil {
			fmt.Println("FAILED")
			return fmt.Errorf("failed to apply migration %s: %w", file, err)
		}
		fmt.Println("OK")
	}

	fmt.Println("Migrations up to date.")
	return nil
}

func (m *Migrator) ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) getAppliedMigrations() (map[string]bool, error) {
	rows, err := m.db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) applyMigration(filename string) error {
	content, err := os.ReadFile(filepath.Join(m.migrationsDir, filename))
	if err != nil {
		return err
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Execute migration SQL
	if _, err := tx.Exec(string(content)); err != nil {
		return err
	}

	// Record migration
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", filename); err != nil {
		return err
	}

	return tx.Commit()
}

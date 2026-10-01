// Package testdb spins up a throwaway Postgres for integration tests.
//
// It prefers TEST_DATABASE_URL (e.g. a CI service container); otherwise it
// starts an embedded Postgres (binaries are downloaded once into
// ~/.embedded-postgres-go). Every test package gets a fresh database with
// the migrations applied.
package testdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"

	"github.com/thanosd/focus/backend/internal/database"
)

var (
	mu       sync.Mutex
	embedded *embeddedpostgres.EmbeddedPostgres
	baseURL  string
	refs     int
)

// Open returns a migrated database. Call the returned cleanup when done.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("FOCUS_SKIP_DB_TESTS") != "" {
		t.Skip("FOCUS_SKIP_DB_TESTS set")
	}
	url := acquire(t)
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	if err := database.NewMigrator(db, migrationsDir()).Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		release()
	})
	return db
}

func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

// acquire starts (or reuses) the embedded server and creates a fresh
// database so parallel test packages don't collide.
func acquire(t *testing.T) string {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	if baseURL == "" {
		if env := os.Getenv("TEST_DATABASE_URL"); env != "" {
			baseURL = env
		} else {
			port := uint32(54329 + os.Getpid()%1000)
			cacheDir, _ := os.UserCacheDir()
			cfg := embeddedpostgres.DefaultConfig().
				Version(embeddedpostgres.V16).
				Port(port).
				Username("postgres").
				Password("postgres").
				Database("postgres").
				RuntimePath(filepath.Join(os.TempDir(), fmt.Sprintf("focus-embedded-pg-%d", os.Getpid()))).
				BinariesPath(filepath.Join(cacheDir, "focus-embedded-pg-bin")).
				StartTimeout(60 * time.Second)
			embedded = embeddedpostgres.NewDatabase(cfg)
			if err := embedded.Start(); err != nil {
				t.Fatalf("start embedded postgres: %v", err)
			}
			baseURL = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/postgres?sslmode=disable", port)
		}
	}
	refs++
	admin, err := sql.Open("postgres", baseURL)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	defer func() { _ = admin.Close() }()
	name := fmt.Sprintf("focus_test_%d_%d", os.Getpid(), time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	return replaceDB(baseURL, name)
}

func release() {
	mu.Lock()
	defer mu.Unlock()
	refs--
	if refs == 0 && embedded != nil {
		_ = embedded.Stop()
		embedded = nil
		baseURL = ""
	}
}

func replaceDB(url, name string) string {
	// url ends with /postgres?sslmode=...; swap the db name.
	i := len(url) - 1
	for i >= 0 && url[i] != '/' {
		i--
	}
	rest := url[i+1:]
	q := ""
	for j := 0; j < len(rest); j++ {
		if rest[j] == '?' {
			q = rest[j:]
			break
		}
	}
	return url[:i+1] + name + q
}

// Command focus is the Focus backend binary. It multiplexes three
// subcommands: server (HTTP API + MCP), worker (Temporal), migrate.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	csrf "filippo.io/csrf/gorilla"
	_ "github.com/lib/pq"

	"github.com/thanosd/focus/backend/internal/adapters"
	"github.com/thanosd/focus/backend/internal/config"
	"github.com/thanosd/focus/backend/internal/database"
	"github.com/thanosd/focus/backend/internal/handlers"
	"github.com/thanosd/focus/backend/internal/mcpserver"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/temporal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "server":
		runServer(cfg)
	case "worker":
		runWorker(cfg)
	case "migrate":
		runMigrate(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: focus <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  server    Start the HTTP API (+ MCP endpoint at /mcp)")
	fmt.Println("  worker    Start the Temporal worker (nightly maintenance)")
	fmt.Println("  migrate   Apply pending database migrations")
}

func openDB(cfg *config.Config) *sql.DB {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}
	fmt.Println("Database connection established")
	return db
}

func runMigrate(cfg *config.Config) {
	if err := config.MustHave("DATABASE_URL"); err != nil && !cfg.IsDevelopment() {
		log.Fatal(err)
	}
	fmt.Printf("Running migrations for %s\n", cfg.ProjectName)
	db := openDB(cfg)
	defer func() { _ = db.Close() }()

	// In Docker the migrations live at /app/migrations; locally we may be
	// at the repo root or inside backend/.
	dir := "migrations"
	for _, candidate := range []string{"/app/migrations", "backend/migrations"} {
		if _, err := os.Stat(candidate); err == nil {
			dir = candidate
			break
		}
	}
	if err := database.NewMigrator(db, dir).Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}

func runServer(cfg *config.Config) {
	fmt.Printf("Starting %s v%s (%s)\n", cfg.ProjectName, cfg.Version, cfg.Environment)
	// Fail fast on anything the server cannot run without. In development
	// the defaults in config.go cover DATABASE_URL and the CSRF key, but
	// Google sign-in and the allowlist have no sane default anywhere.
	required := []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "AUTH_ALLOWED_EMAILS"}
	if !cfg.IsDevelopment() {
		required = append(required, "DATABASE_URL", "CSRF_AUTH_KEY", "BACKEND_CORS_ORIGINS", "FRONTEND_URL", "API_URL")
	}
	if err := config.MustHave(required...); err != nil {
		log.Fatal(err)
	}
	if cfg.ClaudeAPIKey == "" {
		log.Println("CLAUDE_API_KEY not set; natural-language dates use the rule-based parser only")
	} else {
		log.Printf("Claude date parsing enabled (model %s)", cfg.ClaudeModel)
	}

	db := openDB(cfg)
	defer func() { _ = db.Close() }()

	userRepo := adapters.NewUserRepository(db)
	sessionRepo := adapters.NewSessionRepository(db)
	stateRepo := adapters.NewOAuthStateRepository(db)
	tokenRepo := adapters.NewAPITokenRepository(db)
	projectRepo := adapters.NewProjectRepository(db)
	taskRepo := adapters.NewTaskRepository(db)
	tagRepo := adapters.NewTagRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	authService, err := services.NewAuthService(ctx, cfg, userRepo, sessionRepo, stateRepo, tokenRepo)
	cancel()
	if err != nil {
		log.Fatalf("Failed to initialise auth: %v", err)
	}

	var ai dateparse.AIResolver
	if resolver, err := dateparse.NewClaudeResolver(cfg.ClaudeAPIKey, cfg.ClaudeModel); err == nil {
		ai = resolver
	} else if !errors.Is(err, dateparse.ErrAIDisabled) {
		log.Fatalf("Failed to initialise Claude: %v", err)
	}
	parser := dateparse.NewParser(ai)

	taskService := services.NewTaskService(taskRepo, projectRepo, parser)
	projectService := services.NewProjectService(projectRepo, taskRepo)
	tagService := services.NewTagService(tagRepo)

	mcp := mcpserver.New(mcpserver.Deps{
		Auth: authService, Tasks: taskService, Projects: projectService, Tags: tagService, Users: userRepo, Version: cfg.Version,
	})

	router := handlers.NewRouter(handlers.Deps{
		Auth:     handlers.NewAuthHandler(cfg, authService),
		Tasks:    handlers.NewTaskHandler(taskService),
		Projects: handlers.NewProjectHandler(projectService),
		Tags:     handlers.NewTagHandler(tagService),
		MCP:      mcp.Handler(),
		CORS:     handlers.CORSMiddleware(cfg.BackendCORSOrigins),
		CSRF:     csrfMiddleware(cfg),
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		fmt.Printf("Server listening on port %s\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

// csrfMiddleware mirrors team-metrics: filippo.io/csrf inspects Fetch
// metadata (Sec-Fetch-Site) and the Origin header rather than tokens.
// Trusted origins are the CORS origins with their scheme stripped.
func csrfMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	hosts := make([]string, 0, len(cfg.BackendCORSOrigins))
	for _, origin := range cfg.BackendCORSOrigins {
		clean := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
		hosts = append(hosts, clean)
	}
	return csrf.Protect([]byte(cfg.CSRFSecret), csrf.TrustedOrigins(hosts))
}

func runWorker(cfg *config.Config) {
	fmt.Printf("Starting Temporal worker for %s v%s (%s)\n", cfg.ProjectName, cfg.Version, cfg.Environment)
	if err := config.MustHave("TEMPORAL_HOST", "TEMPORAL_NAMESPACE"); err != nil {
		log.Fatal(err)
	}
	db := openDB(cfg)
	defer func() { _ = db.Close() }()

	w, err := worker.New(worker.Config{HostPort: cfg.TemporalHost, Namespace: cfg.TemporalNamespace},
		adapters.NewSessionRepository(db), adapters.NewOAuthStateRepository(db))
	if err != nil {
		log.Fatalf("Error creating worker: %v", err)
	}
	fmt.Printf("Temporal worker connected to %s (namespace %s)\n", cfg.TemporalHost, cfg.TemporalNamespace)
	if err := w.Start(); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}

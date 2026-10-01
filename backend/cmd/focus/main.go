// Command focus is the Focus backend binary. It multiplexes three
// subcommands: server (HTTP API + MCP), worker (Temporal), migrate.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
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
	"github.com/thanosd/focus/backend/internal/domain"
	"github.com/thanosd/focus/backend/internal/handlers"
	"github.com/thanosd/focus/backend/internal/importer"
	"github.com/thanosd/focus/backend/internal/mcpserver"
	"github.com/thanosd/focus/backend/internal/services"
	"github.com/thanosd/focus/backend/internal/services/dateparse"
	"github.com/thanosd/focus/backend/internal/services/repeatparse"
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
	case "import-omnifocus":
		runImportOmniFocus(cfg, os.Args[2:])
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
	fmt.Println("  import-omnifocus <export.csv> --user <email> [--dry-run] [--allow-existing]")
	fmt.Println("            Import an OmniFocus CSV export into that user's account")
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
	var repeatAI repeatparse.AIResolver
	if resolver, err := repeatparse.NewClaudeResolver(cfg.ClaudeAPIKey, cfg.ClaudeModel); err == nil {
		repeatAI = resolver
	}
	repeatParser := repeatparse.NewParser(repeatAI)

	taskService := services.NewTaskService(taskRepo, projectRepo, parser, repeatParser)
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

// runImportOmniFocus loads an OmniFocus CSV export. --dry-run parses and
// prints the plan without touching the database. The user row is created
// (by email) when it doesn't exist yet so the import can run before the
// first sign-in; Google fills in the profile on login.
func runImportOmniFocus(cfg *config.Config, args []string) {
	fs := flag.NewFlagSet("import-omnifocus", flag.ExitOnError)
	userEmail := fs.String("user", "", "email of the account to import into (required)")
	dryRun := fs.Bool("dry-run", false, "parse and print the plan without writing anything")
	allowExisting := fs.Bool("allow-existing", false, "import even if the account already has projects or tasks")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: focus import-omnifocus <export.csv> --user <email> [--dry-run] [--allow-existing]")
		fs.PrintDefaults()
	}
	// Accept the file path before or after the flags.
	var path string
	var rest []string
	for _, a := range args {
		if path == "" && !strings.HasPrefix(a, "-") && strings.HasSuffix(strings.ToLower(a), ".csv") {
			path = a
			continue
		}
		rest = append(rest, a)
	}
	_ = fs.Parse(rest)
	if path == "" && fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	if path == "" {
		fs.Usage()
		os.Exit(2)
	}
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	plan, err := importer.Parse(f)
	if err != nil {
		log.Fatalf("parse %s: %v", path, err)
	}
	fmt.Print(plan.Summary())
	if *dryRun {
		fmt.Println("\nDry run: nothing written.")
		return
	}
	if *userEmail == "" {
		log.Fatal("--user <email> is required (omit it only with --dry-run)")
	}
	if !cfg.IsEmailAllowed(*userEmail) && len(cfg.AuthAllowedEmails) > 0 {
		log.Printf("warning: %s is not in AUTH_ALLOWED_EMAILS; they won't be able to sign in until added", *userEmail)
	}

	db := openDB(cfg)
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	userRepo := adapters.NewUserRepository(db)
	projectRepo := adapters.NewProjectRepository(db)
	taskRepo := adapters.NewTaskRepository(db)
	tagRepo := adapters.NewTagRepository(db)

	user, err := userRepo.GetByEmail(ctx, *userEmail)
	if err != nil {
		log.Fatalf("look up user: %v", err)
	}
	if user == nil {
		if user, err = userRepo.UpsertFromGoogle(ctx, *userEmail, "", "", ""); err != nil {
			log.Fatalf("create user: %v", err)
		}
		fmt.Printf("Created account for %s\n", user.Email)
	}
	if !*allowExisting {
		existing, err := projectRepo.List(ctx, user.ID, []domain.ProjectStatus{domain.ProjectActive, domain.ProjectOnHold, domain.ProjectCompleted, domain.ProjectDropped})
		if err != nil {
			log.Fatalf("check existing projects: %v", err)
		}
		tasks, err := taskRepo.List(ctx, user.ID, domain.TaskFilter{View: domain.ViewAll})
		if err != nil {
			log.Fatalf("check existing tasks: %v", err)
		}
		if len(existing) > 0 || len(tasks) > 0 {
			log.Fatalf("%s already has %d projects and %d active tasks; re-run with --allow-existing to import anyway (this does not de-duplicate)", user.Email, len(existing), len(tasks))
		}
	}

	parser := dateparse.NewParser(nil)
	im := &importer.Importer{
		Projects: services.NewProjectService(projectRepo, taskRepo),
		Tasks:    services.NewTaskService(taskRepo, projectRepo, parser),
		Tags:     services.NewTagService(tagRepo),
		TaskRepo: taskRepo,
	}
	res, err := im.Apply(ctx, user.ID, plan)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}
	fmt.Printf("\nImported %d projects, %d tasks, %d new tags for %s\n", res.Projects, res.Tasks, res.Tags, user.Email)
}

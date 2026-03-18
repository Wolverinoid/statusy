// Package setup handles first-run initialization of Statusy.
// It supports two modes:
//  1. Web setup wizard  — browser-based, triggered when no config exists and
//     no env vars provide a full configuration.
//  2. Env-var / Docker  — skips the wizard when STATUSY_DATABASE_DSN and
//     STATUSY_AUTH_JWT_SECRET are both set; admin is seeded automatically.
package setup

import (
	"bufio"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"log/slog"

	"github.com/statusy/statusy/internal/models"
)

// configPath is where config.yaml is written/read.
// In Docker the working dir is /app, data volume is /app/data — write there so it persists.
const configPath = "data/config.yaml"

// defaultSQLitePath is the default SQLite DB path — inside the data volume.
const defaultSQLitePath = "data/statusy.db"

// ── Setup-needed checks ───────────────────────────────────────────────────────

// IsWebSetupNeeded returns true when the app should start in web-wizard mode:
// no config.yaml exists AND env vars don't provide a full configuration.
func IsWebSetupNeeded() bool {
	_, err := os.Stat(configPath)
	if !errors.Is(err, os.ErrNotExist) {
		return false // config file exists
	}
	if os.Getenv("STATUSY_DATABASE_DSN") != "" && os.Getenv("STATUSY_AUTH_JWT_SECRET") != "" {
		return false // fully configured via env vars
	}
	return true
}

// IsFirstRun is kept for backward compatibility — same logic as IsWebSetupNeeded.
func IsFirstRun() bool { return IsWebSetupNeeded() }

// ── Setup-done channel ────────────────────────────────────────────────────────

var (
	setupDoneCh   = make(chan struct{})
	setupDoneOnce sync.Once
	setupCallback func()
	setupCbMu     sync.Mutex
)

// SetupDone returns a channel that is closed when web setup completes.
func SetupDone() <-chan struct{} { return setupDoneCh }

// OnSetupComplete registers a callback invoked after setup finishes.
func OnSetupComplete(fn func()) {
	setupCbMu.Lock()
	defer setupCbMu.Unlock()
	setupCallback = fn
}

func signalSetupDone() {
	setupDoneOnce.Do(func() {
		setupCbMu.Lock()
		cb := setupCallback
		setupCbMu.Unlock()
		if cb != nil {
			go cb()
		}
		close(setupDoneCh)
	})
}

// ── Web setup router ──────────────────────────────────────────────────────────

// NewSetupRouter returns a Chi router that serves the setup wizard API
// and the embedded React SPA (falling back to index.html for all non-API paths).
func NewSetupRouter(frontendFS embed.FS, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Setup API — no auth required
	r.Get("/api/setup/status", handleSetupStatus)
	r.Post("/api/setup/complete", handleSetupComplete(logger))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","mode":"setup"}`)) //nolint:errcheck
	})

	// Serve embedded SPA — all other routes fall back to index.html
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		logger.Error("failed to sub frontend/dist", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(distFS))
		r.Handle("/*", spaHandler(fileServer, distFS))
	}

	return r
}

func spaHandler(fileServer http.Handler, distFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := distFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

// GET /api/setup/status
func handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	needed := IsWebSetupNeeded()
	json.NewEncoder(w).Encode(map[string]bool{"needed": needed}) //nolint:errcheck
}

// SetupRequest is the payload for POST /api/setup/complete.
type SetupRequest struct {
	// Database
	DBType  string `json:"db_type"`  // sqlite | mysql | postgres
	DBPath  string `json:"db_path"`  // sqlite only
	DBHost  string `json:"db_host"`
	DBPort  string `json:"db_port"`
	DBUser  string `json:"db_user"`
	DBPass  string `json:"db_pass"`
	DBName  string `json:"db_name"`
	SSLMode string `json:"ssl_mode"` // postgres only

	// Admin account
	AdminUsername string `json:"admin_username"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

// POST /api/setup/complete
func handleSetupComplete(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SetupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}

		// Validate
		if err := validateSetupRequest(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
			return
		}

		// Build DSN
		dbType, dsn, err := buildDSN(&req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
			return
		}

		// Connect & migrate
		db, err := openDB(dbType, dsn)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "database connection failed: " + err.Error()}) //nolint:errcheck
			return
		}
		if err := migrate(db); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "migration failed: " + err.Error()}) //nolint:errcheck
			return
		}

		// Create admin user
		hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, `{"error":"failed to hash password"}`, http.StatusInternalServerError)
			return
		}
		admin := models.User{
			Username:    req.AdminUsername,
			Email:       req.AdminEmail,
			Password:    string(hash),
			DisplayName: req.AdminUsername,
			Role:        models.RoleAdmin,
			AuthSource:  models.AuthLocal,
			Active:      true,
		}
		if err := db.Create(&admin).Error; err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to create admin: " + err.Error()}) //nolint:errcheck
			return
		}

		// Generate JWT secret & write config
		jwtSecret, err := generateSecret(48)
		if err != nil {
			http.Error(w, `{"error":"failed to generate secret"}`, http.StatusInternalServerError)
			return
		}
		if err := writeConfig(dbType, dsn, jwtSecret); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to write config: " + err.Error()}) //nolint:errcheck
			return
		}

		logger.Info("web setup complete", "admin", req.AdminUsername, "db_type", dbType)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck

		// Signal main to restart in normal mode
		go signalSetupDone()
	}
}

func validateSetupRequest(req *SetupRequest) error {
	if req.DBType == "" {
		return fmt.Errorf("db_type is required")
	}
	if req.AdminUsername == "" {
		return fmt.Errorf("admin_username is required")
	}
	if req.AdminEmail == "" {
		return fmt.Errorf("admin_email is required")
	}
	if len(req.AdminPassword) < 8 {
		return fmt.Errorf("admin_password must be at least 8 characters")
	}
	return nil
}

func buildDSN(req *SetupRequest) (dbType, dsn string, err error) {
	switch req.DBType {
	case "sqlite":
		path := req.DBPath
		if path == "" {
			path = defaultSQLitePath
		}
		if dir := filepath.Dir(path); dir != "." {
			_ = os.MkdirAll(dir, 0o755)
		}
		return "sqlite", path, nil
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			req.DBUser, req.DBPass, req.DBHost, req.DBPort, req.DBName)
		return "mysql", dsn, nil
	case "postgres":
		sslMode := req.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			req.DBHost, req.DBPort, req.DBUser, req.DBPass, req.DBName, sslMode)
		return "postgres", dsn, nil
	default:
		return "", "", fmt.Errorf("unknown db_type: %q", req.DBType)
	}
}

// ── EnsureAdminExists ─────────────────────────────────────────────────────────

// EnsureAdminExists checks if any admin user exists in the DB.
// If not, it creates one from env vars (STATUSY_ADMIN_USERNAME,
// STATUSY_ADMIN_EMAIL, STATUSY_ADMIN_PASSWORD) or safe defaults.
// This covers Docker/env-var deployments that skip the interactive wizard.
func EnsureAdminExists(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	if count > 0 {
		return nil
	}

	username := getEnvOr("STATUSY_ADMIN_USERNAME", "admin")
	email := getEnvOr("STATUSY_ADMIN_EMAIL", "admin@example.com")
	password := getEnvOr("STATUSY_ADMIN_PASSWORD", "admin")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing admin password: %w", err)
	}

	admin := models.User{
		Username:    username,
		Email:       email,
		Password:    string(hash),
		DisplayName: username,
		Role:        models.RoleAdmin,
		AuthSource:  models.AuthLocal,
		Active:      true,
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("creating admin user: %w", err)
	}

	fmt.Printf("\n✅  Admin user %q created (password from STATUSY_ADMIN_PASSWORD or default \"admin\").\n", username)
	fmt.Println("⚠️   Change the admin password after first login!")
	return nil
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ── Terminal wizard (kept for bare-metal installs) ────────────────────────────

// Run executes the interactive terminal first-run wizard.
// On success it writes config.yaml and returns the opened *gorm.DB.
func Run() (*gorm.DB, error) {
	printBanner()
	reader := bufio.NewReader(os.Stdin)

	dbType, dsn, err := promptDatabase(reader)
	if err != nil {
		return nil, err
	}

	fmt.Println("\n⏳  Connecting to database…")
	db, err := openDB(dbType, dsn)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}
	fmt.Println("✅  Connected.")

	fmt.Println("⏳  Running migrations…")
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	fmt.Println("✅  Schema ready.")

	if err := promptAdminUser(reader, db); err != nil {
		return nil, err
	}

	jwtSecret, err := generateSecret(48)
	if err != nil {
		return nil, fmt.Errorf("generating JWT secret: %w", err)
	}
	if err := writeConfig(dbType, dsn, jwtSecret); err != nil {
		return nil, fmt.Errorf("writing config.yaml: %w", err)
	}

	fmt.Printf("\n✅  config.yaml written.\n")
	fmt.Println("🚀  Starting Statusy…\n")
	return db, nil
}

func promptDatabase(r *bufio.Reader) (dbType, dsn string, err error) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│  Database                               │")
	fmt.Println("├─────────────────────────────────────────┤")
	fmt.Println("│  1) SQLite   (single file, zero config) │")
	fmt.Println("│  2) MySQL / MariaDB                     │")
	fmt.Println("│  3) PostgreSQL                          │")
	fmt.Println("└─────────────────────────────────────────┘")

	choice := prompt(r, "Choose [1-3]", "1")
	switch choice {
	case "1", "sqlite":
		path := prompt(r, "SQLite file path", defaultSQLitePath)
		if dir := filepath.Dir(path); dir != "." {
			_ = os.MkdirAll(dir, 0o755)
		}
		return "sqlite", path, nil
	case "2", "mysql":
		host := prompt(r, "Host", "localhost")
		port := prompt(r, "Port", "3306")
		user := prompt(r, "User", "statusy")
		pass := promptPassword("Password")
		dbName := prompt(r, "Database name", "statusy")
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			user, pass, host, port, dbName)
		return "mysql", dsn, nil
	case "3", "postgres":
		host := prompt(r, "Host", "localhost")
		port := prompt(r, "Port", "5432")
		user := prompt(r, "User", "statusy")
		pass := promptPassword("Password")
		dbName := prompt(r, "Database name", "statusy")
		sslMode := prompt(r, "SSL mode (disable/require/verify-full)", "disable")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, pass, dbName, sslMode)
		return "postgres", dsn, nil
	default:
		return "", "", fmt.Errorf("invalid choice: %q", choice)
	}
}

func promptAdminUser(r *bufio.Reader, db *gorm.DB) error {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│  Admin Account                          │")
	fmt.Println("└─────────────────────────────────────────┘")

	username := prompt(r, "Username", "admin")
	email := prompt(r, "Email", "admin@example.com")

	var password string
	for {
		password = promptPassword("Password (min 8 chars)")
		if len(password) < 8 {
			fmt.Println("  ⚠️  Password must be at least 8 characters.")
			continue
		}
		confirm := promptPassword("Confirm password")
		if password != confirm {
			fmt.Println("  ⚠️  Passwords do not match, try again.")
			continue
		}
		break
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	admin := models.User{
		Username:    username,
		Email:       email,
		Password:    string(hash),
		DisplayName: username,
		Role:        models.RoleAdmin,
		AuthSource:  models.AuthLocal,
		Active:      true,
	}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("creating admin user: %w", err)
	}
	fmt.Printf("\n✅  Admin user %q created.\n", username)
	return nil
}

// ── Config writer ─────────────────────────────────────────────────────────────

const configTemplate = `# Statusy configuration — generated on first run
# Edit this file to change settings. Restart Statusy to apply.

server:
  address: ":3000"

database:
  type: {{ .DBType }}
  dsn: "{{ .DSN }}"

auth:
  jwt_secret: "{{ .JWTSecret }}"
  jwt_expiry_hours: 24
  refresh_expiry_hours: 168
  allow_local_login: true

ldap:
  enabled: false
  url: "ldap://ldap.example.com:389"
  tls: none
  bind_dn: ""
  bind_password: ""
  base_dn: ""
  user_filter: "(uid=%s)"
  attr_email: mail
  attr_display_name: cn
  default_role: user

smtp:
  host: ""
  port: 587
  username: ""
  password: ""
  from: ""
  tls: true

telegram:
  bot_token: ""

metrics:
  enabled: true
  bearer_token: ""
`

type configData struct {
	DBType    string
	DSN       string
	JWTSecret string
}

func writeConfig(dbType, dsn, jwtSecret string) error {
	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return err
	}
	// Ensure the directory exists (e.g. data/)
	if dir := filepath.Dir(configPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating config dir: %w", err)
		}
	}
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, configData{DBType: dbType, DSN: dsn, JWTSecret: jwtSecret})
}

// ── DB helpers ────────────────────────────────────────────────────────────────

func openDB(dbType, dsn string) (*gorm.DB, error) {
	cfg := &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}
	switch dbType {
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(dsn), cfg)
		if err != nil {
			return nil, err
		}
		sqlDB, _ := db.DB()
		sqlDB.Exec("PRAGMA journal_mode=WAL;")
		sqlDB.SetMaxOpenConns(1)
		return db, nil
	case "mysql":
		return gorm.Open(mysql.Open(dsn), cfg)
	case "postgres":
		return gorm.Open(postgres.Open(dsn), cfg)
	default:
		return nil, fmt.Errorf("unknown db type: %s", dbType)
	}
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.JWTBlacklist{},
		&models.Monitor{},
		&models.CheckResult{},
		&models.Notification{},
		&models.StatusPage{},
	)
}

// ── UI helpers ────────────────────────────────────────────────────────────────

func printBanner() {
	fmt.Println()
	fmt.Println("  ███████╗████████╗ █████╗ ████████╗██╗   ██╗███████╗██╗   ██╗")
	fmt.Println("  ██╔════╝╚══██╔══╝██╔══██╗╚══██╔══╝██║   ██║██╔════╝╚██╗ ██╔╝")
	fmt.Println("  ███████╗   ██║   ███████║   ██║   ██║   ██║███████╗ ╚████╔╝ ")
	fmt.Println("  ╚════██║   ██║   ██╔══██║   ██║   ██║   ██║╚════██║  ╚██╔╝  ")
	fmt.Println("  ███████║   ██║   ██║  ██║   ██║   ╚██████╔╝███████║   ██║   ")
	fmt.Println("  ╚══════╝   ╚═╝   ╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚══════╝   ╚═╝   ")
	fmt.Println()
	fmt.Println("  First-run setup wizard")
	fmt.Println("  ─────────────────────────────────────────────────────────────")
}

func prompt(r *bufio.Reader, question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", question, defaultVal)
	} else {
		fmt.Printf("  %s: ", question)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptPassword(question string) string {
	fmt.Printf("  %s: ", question)
	if term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err == nil {
			return string(b)
		}
	}
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func generateSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

package helper

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// TestDB manages test database connection
type TestDB struct {
	DB   *sql.DB
	Name string
}

// SetupTestDB sets up a test database
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Load environment variables - use .env.e2e for E2E tests, .env.test for integration tests
	if os.Getenv("E2E_TEST") == "true" {
		_ = godotenv.Load(".env.e2e")
	} else {
		_ = godotenv.Load(".env.test")
	}

	// Read environment variables
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "root")
	password := getEnv("DB_PASSWORD", "test")

	// Generate unique test database name
	dbName := fmt.Sprintf("test_%d", time.Now().UnixNano())

	// Connect to MySQL server without selecting a database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?multiStatements=true", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// Create test database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		_ = db.Close()
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to the newly created database
	_ = db.Close()
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true", user, password, host, port, dbName)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	testDB := &TestDB{
		DB:   db,
		Name: dbName,
	}

	// Run schema migration
	if err := testDB.Migrate(); err != nil {
		testDB.Cleanup()
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return testDB
}

// Migrate applies schema to test database
func (tdb *TestDB) Migrate() error {
	// Apply all migrations in order
	migrations := []string{
		"000001_initial_schema.up.sql",
		"000002_add_deployment_locks.up.sql",
		"000003_add_verification_tokens.up.sql",
		"000004_move_fqdn_to_networks.up.sql",
		"000005_add_initial_templates.up.sql",
		"000006_refactor_deployment_mechanism.up.sql",
		"000007_add_active_deployment_tracking.up.sql",
		"000008_add_volume_slug.up.sql",
		"000009_add_github_installations.up.sql",
		"000010_add_installation_status.up.sql",
		"000011_create_oauth_states.up.sql",
		"000012_update_slug_columns.up.sql",
		"000013_fix_container_slug_index.up.sql",
	}

	for _, migration := range migrations {
		schemaPath := filepath.Join("..", "..", "migration", migration)
		schema, err := os.ReadFile(schemaPath)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", migration, err)
		}

		// Execute schema
		_, err = tdb.DB.Exec(string(schema))
		if err != nil {
			// Log the error with more context
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}
	}

	return nil
}

// TruncateAllTables deletes data from all tables
func (tdb *TestDB) TruncateAllTables() error {
	tables := []string{
		"USERS",
		// Add other tables as needed
	}

	// Disable foreign key checks
	if _, err := tdb.DB.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}

	// Truncate each table
	for _, table := range tables {
		if _, err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			return err
		}
	}

	// Re-enable foreign key checks
	if _, err := tdb.DB.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return err
	}

	return nil
}

// Cleanup drops the test database and closes connection
func (tdb *TestDB) Cleanup() {
	if tdb.DB != nil {
		_, _ = tdb.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.Name))
		_ = tdb.DB.Close()
	}
}

// GetUserByUsername retrieves a user by username from the database
func (tdb *TestDB) GetUserByUsername(username string) (map[string]interface{}, error) {
	query := "SELECT user_id, username, email, password_hash, status, created_at, updated_at FROM USERS WHERE username = ?"
	row := tdb.DB.QueryRow(query, username)

	var userID uint
	var usernameVal, email, passwordHash, status string
	var createdAt, updatedAt time.Time

	err := row.Scan(&userID, &usernameVal, &email, &passwordHash, &status, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"user_id":       userID,
		"username":      usernameVal,
		"email":         email,
		"password_hash": passwordHash,
		"status":        status,
		"created_at":    createdAt,
		"updated_at":    updatedAt,
	}, nil
}

// getEnv retrieves environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

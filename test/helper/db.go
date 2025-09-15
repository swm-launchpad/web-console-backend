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

	// Load environment variables
	godotenv.Load(".env.test")

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
		db.Close()
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to the newly created database
	db.Close()
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true", user, password, host, port, dbName)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
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
	// Read schema file
	schemaPath := filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute schema
	_, err = tdb.DB.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
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

	// Truncate all tables
	for _, table := range tables {
		if _, err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			return err
		}
	}

	// Enable foreign key checks
	if _, err := tdb.DB.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return err
	}

	return nil
}

// Cleanup cleans up test database
func (tdb *TestDB) Cleanup() {
	if tdb.DB != nil {
		// Drop database
		tdb.DB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.Name))
		tdb.DB.Close()
	}
}

// BeginTx starts a transaction
func (tdb *TestDB) BeginTx() (*sql.Tx, error) {
	return tdb.DB.Begin()
}

// InsertTestUser inserts a test user
func (tdb *TestDB) InsertTestUser(username, passwordHash, email string) (uint, error) {
	query := `
		INSERT INTO USERS (username, password_hash, email, status, is_deleted, created_at)
		VALUES (?, ?, ?, 'active', false, NOW())
	`

	result, err := tdb.DB.Exec(query, username, passwordHash, email)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint(id), nil
}

// GetUserByUsername retrieves user by username
func (tdb *TestDB) GetUserByUsername(username string) (map[string]interface{}, error) {
	query := `
		SELECT user_id, username, password_hash, email, status, is_deleted, created_at
		FROM USERS
		WHERE username = ?
	`

	row := tdb.DB.QueryRow(query, username)

	var user = make(map[string]interface{})
	var userID uint
	var uname, passwordHash, status string
	var email sql.NullString
	var isDeleted bool
	var createdAt time.Time

	err := row.Scan(&userID, &uname, &passwordHash, &email, &status, &isDeleted, &createdAt)
	if err != nil {
		return nil, err
	}

	user["user_id"] = userID
	user["username"] = uname
	user["password_hash"] = passwordHash
	if email.Valid {
		user["email"] = email.String
	}
	user["status"] = status
	user["is_deleted"] = isDeleted
	user["created_at"] = createdAt

	return user, nil
}

// getEnv reads environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
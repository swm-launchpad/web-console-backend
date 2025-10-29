package settings

import (
	"context"
	"database/sql"
	"fmt"
)

// SettingsRepository defines the interface for settings data access
type SettingsRepository interface {
	// GetByKey retrieves a setting value by key
	GetByKey(ctx context.Context, key string) (*Setting, error)

	// GetByCategory retrieves all settings in a category
	GetByCategory(ctx context.Context, category string) ([]*Setting, error)

	// GetAll retrieves all settings
	GetAll(ctx context.Context) ([]*Setting, error)

	// Update updates a setting value (only if is_editable = true)
	Update(ctx context.Context, key, value string, updatedBy *int) error
}

// Setting represents a system setting
type Setting struct {
	Key         string
	Value       string
	ValueType   string
	Category    string
	Description string
	IsEditable  bool
	UpdatedBy   *int
	CreatedAt   string
	UpdatedAt   string
}

// settingsRepository is the MySQL implementation of SettingsRepository
type settingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new instance of SettingsRepository
func NewSettingsRepository(db *sql.DB) SettingsRepository {
	return &settingsRepository{
		db: db,
	}
}

// GetByKey retrieves a setting value by key
func (r *settingsRepository) GetByKey(ctx context.Context, key string) (*Setting, error) {
	query := `
		SELECT setting_key, setting_value, value_type, category,
		       COALESCE(description, ''), is_editable, updated_by,
		       created_at, updated_at
		FROM SYSTEM_SETTINGS
		WHERE setting_key = ?
	`

	var setting Setting
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&setting.Key,
		&setting.Value,
		&setting.ValueType,
		&setting.Category,
		&setting.Description,
		&setting.IsEditable,
		&setting.UpdatedBy,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	return &setting, nil
}

// GetByCategory retrieves all settings in a category
func (r *settingsRepository) GetByCategory(ctx context.Context, category string) ([]*Setting, error) {
	query := `
		SELECT setting_key, setting_value, value_type, category,
		       COALESCE(description, ''), is_editable, updated_by,
		       created_at, updated_at
		FROM SYSTEM_SETTINGS
		WHERE category = ?
		ORDER BY setting_key
	`

	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings by category: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var settings []*Setting
	for rows.Next() {
		var s Setting
		err := rows.Scan(
			&s.Key,
			&s.Value,
			&s.ValueType,
			&s.Category,
			&s.Description,
			&s.IsEditable,
			&s.UpdatedBy,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, &s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}

	return settings, nil
}

// GetAll retrieves all settings
func (r *settingsRepository) GetAll(ctx context.Context) ([]*Setting, error) {
	query := `
		SELECT setting_key, setting_value, value_type, category,
		       COALESCE(description, ''), is_editable, updated_by,
		       created_at, updated_at
		FROM SYSTEM_SETTINGS
		ORDER BY category, setting_key
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all settings: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var settings []*Setting
	for rows.Next() {
		var s Setting
		err := rows.Scan(
			&s.Key,
			&s.Value,
			&s.ValueType,
			&s.Category,
			&s.Description,
			&s.IsEditable,
			&s.UpdatedBy,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, &s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}

	return settings, nil
}

// Update updates a setting value (only if is_editable = true)
func (r *settingsRepository) Update(ctx context.Context, key, value string, updatedBy *int) error {
	// Check if setting is editable
	checkQuery := `
		SELECT is_editable FROM SYSTEM_SETTINGS WHERE setting_key = ?
	`

	var isEditable bool
	err := r.db.QueryRowContext(ctx, checkQuery, key).Scan(&isEditable)
	if err == sql.ErrNoRows {
		return fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return fmt.Errorf("failed to check setting editability: %w", err)
	}

	if !isEditable {
		return fmt.Errorf("setting '%s' is not editable", key)
	}

	// Update the setting
	updateQuery := `
		UPDATE SYSTEM_SETTINGS
		SET setting_value = ?, updated_by = ?, updated_at = CURRENT_TIMESTAMP
		WHERE setting_key = ?
	`

	result, err := r.db.ExecContext(ctx, updateQuery, value, updatedBy, key)
	if err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("setting not found: %s", key)
	}

	return nil
}

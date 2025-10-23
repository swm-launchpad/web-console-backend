package integration

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

func TestMigration_BuildVars(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	t.Run("BUILD_VARS table exists with correct schema", func(t *testing.T) {
		// Check table exists
		var tableName string
		err := testDB.DB.QueryRow("SHOW TABLES LIKE 'BUILD_VARS'").Scan(&tableName)
		require.NoError(t, err)
		assert.Equal(t, "BUILD_VARS", tableName)

		// Check columns
		rows, err := testDB.DB.Query("DESCRIBE BUILD_VARS")
		require.NoError(t, err)
		defer func() {
			require.NoError(t, rows.Close())
		}()

		columns := make(map[string]string)
		for rows.Next() {
			var field, typ, null, key, def, extra sql.NullString
			err := rows.Scan(&field, &typ, &null, &key, &def, &extra)
			require.NoError(t, err)
			columns[field.String] = typ.String
		}

		// Verify columns
		assert.Contains(t, columns, "build_var_id")
		assert.Contains(t, columns, "container_id")
		assert.Contains(t, columns, "key")
		assert.Contains(t, columns, "value")
		assert.Contains(t, columns, "created_at")
		assert.Contains(t, columns, "updated_at")

		// Verify data types
		assert.Contains(t, columns["build_var_id"], "int")
		assert.Contains(t, columns["container_id"], "int")
		assert.Equal(t, "varchar(255)", columns["key"])
		assert.Equal(t, "text", columns["value"])
	})

	t.Run("BUILD_VARS unique constraint works", func(t *testing.T) {
		// Create test project (slug must be 23 chars: p{timestamp}{random})
		_, err := testDB.DB.Exec(`
			INSERT INTO PROJECTS (name, slug, status, project_operation_status)
			VALUES ('Test Project', 'p20251023000000test001', 'active', 'nothing')
		`)
		require.NoError(t, err)

		// Create test container (slug must be 23 chars: c{timestamp}{random})
		result, err := testDB.DB.Exec(`
			INSERT INTO CONTAINERS (project_id, name, slug)
			SELECT project_id, 'Test Container', 'c20251023000000test001'
			FROM PROJECTS WHERE slug = 'p20251023000000test001'
		`)
		require.NoError(t, err)
		containerID, _ := result.LastInsertId()

		// Insert first build var
		_, err = testDB.DB.Exec(`
			INSERT INTO BUILD_VARS (container_id, `+"`key`"+`, value)
			VALUES (?, 'TEST_KEY', 'test_value')
		`, containerID)
		require.NoError(t, err)

		// Try to insert duplicate key - should fail
		_, err = testDB.DB.Exec(`
			INSERT INTO BUILD_VARS (container_id, `+"`key`"+`, value)
			VALUES (?, 'TEST_KEY', 'another_value')
		`, containerID)
		assert.Error(t, err, "Duplicate key should fail")
	})

	t.Run("BUILD_VARS cascade delete works", func(t *testing.T) {
		// Create test project (slug must be 23 chars)
		_, err := testDB.DB.Exec(`
			INSERT INTO PROJECTS (name, slug, status, project_operation_status)
			VALUES ('Test Project 2', 'p20251023000000test002', 'active', 'nothing')
		`)
		require.NoError(t, err)

		// Create test container (slug must be 23 chars)
		result, err := testDB.DB.Exec(`
			INSERT INTO CONTAINERS (project_id, name, slug)
			SELECT project_id, 'Test Container 2', 'c20251023000000test002'
			FROM PROJECTS WHERE slug = 'p20251023000000test002'
		`)
		require.NoError(t, err)
		containerID, _ := result.LastInsertId()

		// Insert build var
		_, err = testDB.DB.Exec(`
			INSERT INTO BUILD_VARS (container_id, `+"`key`"+`, value)
			VALUES (?, 'CASCADE_TEST', 'value')
		`, containerID)
		require.NoError(t, err)

		// Delete container
		_, err = testDB.DB.Exec("DELETE FROM CONTAINERS WHERE container_id = ?", containerID)
		require.NoError(t, err)

		// Verify build vars were cascade deleted
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM BUILD_VARS WHERE container_id = ?", containerID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Build vars should be cascade deleted")
	})
}

func TestMigration_ContainersModification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	t.Run("CONTAINERS table has needs_build column", func(t *testing.T) {
		// Check columns
		rows, err := testDB.DB.Query("DESCRIBE CONTAINERS")
		require.NoError(t, err)
		defer func() {
			require.NoError(t, rows.Close())
		}()

		columns := make(map[string]string)
		for rows.Next() {
			var field, typ, null, key, def, extra sql.NullString
			err := rows.Scan(&field, &typ, &null, &key, &def, &extra)
			require.NoError(t, err)
			columns[field.String] = typ.String
		}

		// Verify needs_build exists
		assert.Contains(t, columns, "needs_build")
		assert.Contains(t, columns["needs_build"], "tinyint")

		// Verify git_commit_hash still exists (kept for backward compatibility)
		assert.Contains(t, columns, "git_commit_hash")

		// Verify last_built_git_commit_hash still exists
		assert.Contains(t, columns, "last_built_git_commit_hash")
	})

	t.Run("CONTAINERS needs_build default value is TRUE", func(t *testing.T) {
		// Create test project (slug must be 23 chars)
		_, err := testDB.DB.Exec(`
			INSERT INTO PROJECTS (name, slug, status, project_operation_status)
			VALUES ('Test Project 3', 'p20251023000000test003', 'active', 'nothing')
		`)
		require.NoError(t, err)

		// Create test container without specifying needs_build (slug must be 23 chars)
		result, err := testDB.DB.Exec(`
			INSERT INTO CONTAINERS (project_id, name, slug)
			SELECT project_id, 'Test Container 3', 'c20251023000000test003'
			FROM PROJECTS WHERE slug = 'p20251023000000test003'
		`)
		require.NoError(t, err)
		containerID, _ := result.LastInsertId()

		// Check default value
		var needsBuild bool
		err = testDB.DB.QueryRow("SELECT needs_build FROM CONTAINERS WHERE container_id = ?", containerID).Scan(&needsBuild)
		require.NoError(t, err)
		assert.True(t, needsBuild, "Default value of needs_build should be TRUE")
	})
}

func TestMigration_BuildHistoryRefactor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	t.Run("BUILD_HISTORY table has new structure", func(t *testing.T) {
		// Check columns
		rows, err := testDB.DB.Query("DESCRIBE BUILD_HISTORY")
		require.NoError(t, err)
		defer func() {
			require.NoError(t, rows.Close())
		}()

		columns := make(map[string]string)
		for rows.Next() {
			var field, typ, null, key, def, extra sql.NullString
			err := rows.Scan(&field, &typ, &null, &key, &def, &extra)
			require.NoError(t, err)
			columns[field.String] = typ.String
		}

		// Verify new columns exist
		assert.Contains(t, columns, "tekton_event_id")
		assert.Contains(t, columns, "tekton_pipeline_run_name")
		assert.Equal(t, "varchar(255)", columns["tekton_event_id"])
		assert.Equal(t, "varchar(255)", columns["tekton_pipeline_run_name"])

		// Verify old tekton_ref column does not exist
		assert.NotContains(t, columns, "tekton_ref")

		// Verify git_commit_hash still exists
		assert.Contains(t, columns, "git_commit_hash")
	})

	t.Run("BUILD_HISTORY status enum includes new values", func(t *testing.T) {
		// Get enum values
		var columnType string
		err := testDB.DB.QueryRow(`
			SELECT COLUMN_TYPE FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = 'BUILD_HISTORY'
			AND COLUMN_NAME = 'status'
		`).Scan(&columnType)
		require.NoError(t, err)

		// Check that new status values are included
		assert.Contains(t, columnType, "untracked")
		assert.Contains(t, columnType, "backend_trigger_failed")
		assert.Contains(t, columnType, "backend_tracking_failed")
		assert.Contains(t, columnType, "backend_tracking_lost")
		assert.Contains(t, columnType, "running")
		assert.Contains(t, columnType, "success")
		assert.Contains(t, columnType, "failed")
		assert.Contains(t, columnType, "cancelled")
		assert.Contains(t, columnType, "skipped")
	})

	t.Run("BUILD_HISTORY default status is untracked", func(t *testing.T) {
		// Create test project (slug must be 23 chars)
		_, err := testDB.DB.Exec(`
			INSERT INTO PROJECTS (name, slug, status, project_operation_status)
			VALUES ('Test Project 4', 'p20251023000000test004', 'active', 'nothing')
		`)
		require.NoError(t, err)

		// Create test container (slug must be 23 chars)
		result, err := testDB.DB.Exec(`
			INSERT INTO CONTAINERS (project_id, name, slug)
			SELECT project_id, 'Test Container 4', 'c20251023000000test004'
			FROM PROJECTS WHERE slug = 'p20251023000000test004'
		`)
		require.NoError(t, err)
		containerID, _ := result.LastInsertId()

		// Create build history without status
		_, err = testDB.DB.Exec(`
			INSERT INTO BUILD_HISTORY (container_id)
			VALUES (?)
		`, containerID)
		require.NoError(t, err)

		// Check default value
		var status string
		err = testDB.DB.QueryRow("SELECT status FROM BUILD_HISTORY WHERE container_id = ?", containerID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "untracked", status, "Default status should be 'untracked'")
	})
}

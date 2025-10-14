package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectinfra "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

// createTestProject creates a test project for integration tests
func createTestProject(t *testing.T, testDB *helper.TestDB) uint {
	t.Helper()

	query := `
		INSERT INTO PROJECTS (name, slug, status, project_operation_status, cpu_limit, memory_limit, disk_limit, traffic_limit, created_at)
		VALUES (?, ?, 'active', 'nothing', 1000, 2048, 2048, 1048576, NOW())
	`

	// Use nanosecond timestamp for unique slugs
	slug := fmt.Sprintf("test-project-%d", time.Now().UnixNano())
	result, err := testDB.DB.Exec(query, "Test Project", slug)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	return uint(id)
}

func TestDeploymentRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	// Create repository
	repo := projectinfra.NewDeploymentRepository(testDB.DB)
	ctx := context.Background()

	t.Run("Create - New deployment", func(t *testing.T) {
		// Given
		pid := createTestProject(t, testDB)
		d := deployment.NewDeployment(pid)

		// When
		err := repo.Create(ctx, d)

		// Then
		require.NoError(t, err)
		assert.NotZero(t, d.DeploymentID(), "Deployment ID should be set after creation")
		assert.Equal(t, deployment.DeploymentStatusUntracked, d.Status())
	})

	t.Run("Save - Update deployment status", func(t *testing.T) {
		// Given - Create deployment first
		pid := createTestProject(t, testDB)
		d := deployment.NewDeployment(pid)
		err := repo.Create(ctx, d)
		require.NoError(t, err)

		// When - Start deployment
		tektonRef := "pipeline-run-123"
		err = d.MarkAsRunning(tektonRef)
		require.NoError(t, err)
		err = repo.Save(ctx, d)

		// Then
		require.NoError(t, err)

		// Verify by fetching
		fetched, err := repo.FindByID(ctx, d.DeploymentID())
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusRunning, fetched.Status())
		assert.Equal(t, tektonRef, fetched.TektonPipelineRunName())
		assert.False(t, fetched.StartedAt().IsZero())
	})

	t.Run("Save - Not found error", func(t *testing.T) {
		// Given - Create deployment but don't persist it
		d := deployment.NewDeployment(123)
		d.SetDeploymentID(99999) // Non-existent ID

		// When - Try to save
		err := repo.Save(ctx, d)

		// Then - Should return not found error
		assert.ErrorIs(t, err, projecterrors.ErrDeploymentNotFound)
	})

	t.Run("Save - Idempotent update with same values", func(t *testing.T) {
		// Given - Create and start deployment
		pid := createTestProject(t, testDB)
		d := deployment.NewDeployment(pid)
		err := repo.Create(ctx, d)
		require.NoError(t, err)

		tektonRef := "pipeline-run-456"
		err = d.MarkAsRunning(tektonRef)
		require.NoError(t, err)
		err = repo.Save(ctx, d)
		require.NoError(t, err)

		// When - Save again with same values (idempotent retry)
		err = repo.Save(ctx, d)

		// Then - Should succeed (no error, even though 0 rows affected)
		require.NoError(t, err)

		// Verify data is still correct
		fetched, err := repo.FindByID(ctx, d.DeploymentID())
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusRunning, fetched.Status())
		assert.Equal(t, tektonRef, fetched.TektonPipelineRunName())
	})

	t.Run("FindByID - Existing deployment", func(t *testing.T) {
		// Given
		pid := createTestProject(t, testDB)
		d := deployment.NewDeployment(pid)
		err := repo.Create(ctx, d)
		require.NoError(t, err)

		// When
		fetched, err := repo.FindByID(ctx, d.DeploymentID())

		// Then
		require.NoError(t, err)
		assert.Equal(t, d.DeploymentID(), fetched.DeploymentID())
		assert.Equal(t, d.ProjectID(), fetched.ProjectID())
		assert.Equal(t, d.Status(), fetched.Status())
	})

	t.Run("FindByID - Not found", func(t *testing.T) {
		// When
		_, err := repo.FindByID(ctx, 99999)

		// Then
		assert.ErrorIs(t, err, projecterrors.ErrDeploymentNotFound)
	})

	t.Run("FindLatestByProjectID - Returns newest deployment", func(t *testing.T) {
		// Given - Create multiple deployments for the same project
		pid := createTestProject(t, testDB)

		d1 := deployment.NewDeployment(pid)
		err := repo.Create(ctx, d1)
		require.NoError(t, err)

		d2 := deployment.NewDeployment(pid)
		err = repo.Create(ctx, d2)
		require.NoError(t, err)

		d3 := deployment.NewDeployment(pid)
		err = repo.Create(ctx, d3)
		require.NoError(t, err)

		// When
		latest, err := repo.FindLatestByProjectID(ctx, pid)

		// Then
		require.NoError(t, err)
		// With ORDER BY created_at DESC, deployment_id DESC, the latest created (highest ID) is returned
		assert.Equal(t, d3.DeploymentID(), latest.DeploymentID(), "Should return the most recent deployment")
	})

	t.Run("FindLatestByProjectID - Not found", func(t *testing.T) {
		// Given
		pid := createTestProject(t, testDB)

		// When
		_, err := repo.FindLatestByProjectID(ctx, pid)

		// Then
		assert.ErrorIs(t, err, projecterrors.ErrDeploymentNotFound)
	})

	t.Run("FindByProjectID - Pagination", func(t *testing.T) {
		// Given - Create 5 deployments
		pid := createTestProject(t, testDB)

		deploymentIDs := make([]uint, 0, 5)
		for i := 0; i < 5; i++ {
			d := deployment.NewDeployment(pid)
			err := repo.Create(ctx, d)
			require.NoError(t, err)
			deploymentIDs = append(deploymentIDs, d.DeploymentID())
		}

		// When - Fetch first page (limit 3, offset 0)
		page1, err := repo.FindByProjectID(ctx, pid, 3, 0)

		// Then
		require.NoError(t, err)
		assert.Len(t, page1, 3, "First page should have 3 items")

		// When - Fetch second page (limit 3, offset 3)
		page2, err := repo.FindByProjectID(ctx, pid, 3, 3)

		// Then
		require.NoError(t, err)
		assert.Len(t, page2, 2, "Second page should have 2 items")

		// Verify order (newest first by deployment_id DESC) - last created should be first
		assert.Equal(t, deploymentIDs[4], page1[0].DeploymentID(), "Newest deployment should be first")
		assert.Equal(t, deploymentIDs[3], page1[1].DeploymentID())
		assert.Equal(t, deploymentIDs[2], page1[2].DeploymentID())
	})

	t.Run("FindByProjectID - Empty result", func(t *testing.T) {
		// Given
		pid := createTestProject(t, testDB)

		// When
		deployments, err := repo.FindByProjectID(ctx, pid, 10, 0)

		// Then
		require.NoError(t, err)
		assert.Empty(t, deployments)
	})

	t.Run("Full deployment lifecycle", func(t *testing.T) {
		// Given - Create deployment
		pid := createTestProject(t, testDB)
		d := deployment.NewDeployment(pid)
		err := repo.Create(ctx, d)
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusUntracked, d.Status())

		// When - Start deployment
		err = d.MarkAsRunning("pipeline-run-123")
		require.NoError(t, err)
		err = repo.Save(ctx, d)
		require.NoError(t, err)

		// Then - Verify running status
		fetched, err := repo.FindByID(ctx, d.DeploymentID())
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusRunning, fetched.Status())
		assert.Equal(t, "pipeline-run-123", fetched.TektonPipelineRunName())

		// When - Complete deployment
		err = fetched.Complete("Deployment successful")
		require.NoError(t, err)
		err = repo.Save(ctx, fetched)
		require.NoError(t, err)

		// Then - Verify success status
		final, err := repo.FindByID(ctx, d.DeploymentID())
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusSuccess, final.Status())
		assert.Equal(t, "Deployment successful", final.Summary())
		assert.False(t, final.FinishedAt().IsZero())
		assert.True(t, final.IsCompleted())
	})

	t.Run("Failed deployment lifecycle", func(t *testing.T) {
		// Given - Create and start deployment
		pid := createTestProject(t, testDB)
		d := deployment.NewDeployment(pid)
		err := repo.Create(ctx, d)
		require.NoError(t, err)

		err = d.MarkAsRunning("pipeline-run-456")
		require.NoError(t, err)
		err = repo.Save(ctx, d)
		require.NoError(t, err)

		// When - Fail deployment
		fetched, err := repo.FindByID(ctx, d.DeploymentID())
		require.NoError(t, err)
		err = fetched.Fail("Build failed: syntax error")
		require.NoError(t, err)
		err = repo.Save(ctx, fetched)
		require.NoError(t, err)

		// Then - Verify failed status
		final, err := repo.FindByID(ctx, d.DeploymentID())
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusFailed, final.Status())
		assert.Equal(t, "Build failed: syntax error", final.Summary())
		assert.False(t, final.FinishedAt().IsZero())
		assert.True(t, final.IsCompleted())
	})

	t.Run("Cancelled deployment lifecycle", func(t *testing.T) {
		// Given - Create deployment
		pid := createTestProject(t, testDB)
		d := deployment.NewDeployment(pid)
		err := repo.Create(ctx, d)
		require.NoError(t, err)

		// When - Mark trigger failed immediately
		err = d.MarkAsTriggerFailed("Failed to trigger")
		require.NoError(t, err)
		err = repo.Save(ctx, d)
		require.NoError(t, err)

		// Then - Verify backend_trigger_failed status
		fetched, err := repo.FindByID(ctx, d.DeploymentID())
		require.NoError(t, err)
		assert.Equal(t, deployment.DeploymentStatusBackendTriggerFailed, fetched.Status())
		assert.Equal(t, "Failed to trigger", fetched.Summary())
		assert.True(t, fetched.IsCompleted())
	})
}

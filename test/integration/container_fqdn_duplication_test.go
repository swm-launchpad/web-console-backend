package integration

import (
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

// generateValidProjectSlug generates a valid project slug for testing
// Format: p{timestamp}{random} (23 chars total)
func generateValidProjectSlug() string {
	timestamp := time.Now().Format("20060102150405")
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	random := make([]byte, 8)
	for i := range random {
		random[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("p%s%s", timestamp, string(random))
}

// TestFQDNDuplication_SameProject tests FQDN reuse within the same project
func TestFQDNDuplication_SameProject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	fqdn := "test-same-project.launchpad.kr"

	t.Run("Should allow FQDN reuse after soft delete in same project", func(t *testing.T) {
		// Given: Create project and container
		projectID := createTestProjectForFQDN(t, testDB, "test-project-1")
		container1ID := createTestContainerForFQDN(t, testDB, projectID, "container-1")
		network1ID := createTestNetworkForFQDN(t, testDB, container1ID, fqdn, 8080)

		// When: Soft delete the network
		softDeleteNetwork(t, testDB, network1ID)

		// And: Create another container with the same FQDN in the same project
		container2ID := createTestContainerForFQDN(t, testDB, projectID, "container-2")
		network2ID := createTestNetworkForFQDN(t, testDB, container2ID, fqdn, 8081)

		// Then: Should succeed (no error)
		assert.NotZero(t, network2ID)

		// Verify both networks exist with same FQDN (one deleted, one active)
		count := countNetworksWithFQDN(t, testDB, fqdn)
		assert.Equal(t, 2, count, "Should have 2 networks with same FQDN (1 deleted, 1 active)")

		// Verify the active network is the second one
		activeFQDN := getActiveNetworkFQDN(t, testDB, network2ID)
		assert.Equal(t, fqdn, activeFQDN)
	})

	t.Run("Should NOT allow duplicate active FQDN in same project", func(t *testing.T) {
		// Given: Create project and container with FQDN
		projectID := createTestProjectForFQDN(t, testDB, "test-project-2")
		container1ID := createTestContainerForFQDN(t, testDB, projectID, "container-1")
		fqdn2 := "test-active-dup.launchpad.kr"
		createTestNetworkForFQDN(t, testDB, container1ID, fqdn2, 8080)

		// When: Try to create another container with the same FQDN (both active)
		container2ID := createTestContainerForFQDN(t, testDB, projectID, "container-2")

		// Then: Application layer should prevent this (tested in application layer tests)
		// For DB level, we just verify the query behavior
		exists := checkFQDNExistsForProject(t, testDB, fqdn2, projectID)
		assert.True(t, exists, "Should detect duplicate FQDN in same project")

		// Cleanup
		_ = container2ID
	})
}

// TestFQDNDuplication_DifferentProjects tests FQDN isolation between projects
func TestFQDNDuplication_DifferentProjects(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	fqdn := "test-diff-project.launchpad.kr"

	t.Run("Should NOT allow FQDN reuse from soft-deleted network in different project", func(t *testing.T) {
		// Given: Project A has a network with FQDN, then soft deletes it
		projectAID := createTestProjectForFQDN(t, testDB, "project-a")
		containerAID := createTestContainerForFQDN(t, testDB, projectAID, "container-a")
		networkAID := createTestNetworkForFQDN(t, testDB, containerAID, fqdn, 8080)
		softDeleteNetwork(t, testDB, networkAID)

		// When: Project B tries to use the same FQDN
		projectBID := createTestProjectForFQDN(t, testDB, "project-b")

		// Then: Should detect conflict (FQDN still owned by Project A)
		exists := checkFQDNExistsForProject(t, testDB, fqdn, projectBID)
		assert.True(t, exists, "Should detect FQDN conflict from different project even if soft-deleted")
	})

	t.Run("Should allow FQDN reuse after project CASCADE DELETE", func(t *testing.T) {
		// Given: Project C has a network with FQDN
		fqdn2 := "test-cascade.launchpad.kr"
		projectCID := createTestProjectForFQDN(t, testDB, "project-c")
		containerCID := createTestContainerForFQDN(t, testDB, projectCID, "container-c")
		createTestNetworkForFQDN(t, testDB, containerCID, fqdn2, 8080)

		// When: Delete the entire project (CASCADE DELETE)
		deleteProject(t, testDB, projectCID)

		// And: Project D tries to use the same FQDN
		projectDID := createTestProjectForFQDN(t, testDB, "project-d")

		// Then: Should succeed (FQDN released after CASCADE DELETE)
		exists := checkFQDNExistsForProject(t, testDB, fqdn2, projectDID)
		assert.False(t, exists, "Should NOT detect FQDN conflict after project CASCADE DELETE")
	})
}

// TestFQDNDuplication_SoftDeletePreservation tests that FQDN is preserved during soft delete
func TestFQDNDuplication_SoftDeletePreservation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	fqdn := "test-preservation.launchpad.kr"

	t.Run("Should preserve FQDN value during soft delete", func(t *testing.T) {
		// Given: Create network with FQDN
		projectID := createTestProjectForFQDN(t, testDB, "test-project")
		containerID := createTestContainerForFQDN(t, testDB, projectID, "test-container")
		networkID := createTestNetworkForFQDN(t, testDB, containerID, fqdn, 8080)

		// When: Soft delete the network
		softDeleteNetwork(t, testDB, networkID)

		// Then: FQDN should still be in database (not NULL)
		deletedFQDN := getDeletedNetworkFQDN(t, testDB, networkID)
		assert.Equal(t, fqdn, deletedFQDN, "FQDN should be preserved after soft delete")
	})
}

// Helper functions

func createTestProjectForFQDN(t *testing.T, testDB *helper.TestDB, name string) uint {
	t.Helper()

	slug := generateValidProjectSlug()
	query := `
		INSERT INTO PROJECTS (name, slug, status, project_operation_status, cpu_limit, memory_limit, disk_limit, traffic_limit, created_at)
		VALUES (?, ?, 'active', 'nothing', 1000, 2048, 2048, 1048576, NOW())
	`
	result, err := testDB.DB.Exec(query, name, slug)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	return uint(id)
}

func createTestContainerForFQDN(t *testing.T, testDB *helper.TestDB, projectID uint, name string) uint {
	t.Helper()

	// Generate unique slug for container
	slug := fmt.Sprintf("c%d%d", time.Now().Unix(), rand.Intn(1000))

	query := `
		INSERT INTO CONTAINERS (
			project_id, name, slug, template_id, cpu_limit, memory_limit, created_at
		)
		VALUES (?, ?, ?, 1, 1000, 2048, NOW())
	`
	result, err := testDB.DB.Exec(query, projectID, name, slug)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	return uint(id)
}

func createTestNetworkForFQDN(t *testing.T, testDB *helper.TestDB, containerID uint, fqdn string, internalPort int) uint {
	t.Helper()

	query := `
		INSERT INTO NETWORKS (container_id, fqdn, internal_port, type, created_at)
		VALUES (?, ?, ?, 'http', NOW())
	`
	result, err := testDB.DB.Exec(query, containerID, fqdn, internalPort)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	return uint(id)
}

func softDeleteNetwork(t *testing.T, testDB *helper.TestDB, networkID uint) {
	t.Helper()

	query := `
		UPDATE NETWORKS SET
			is_deleted = TRUE,
			deleted_at = ?
		WHERE network_id = ?
	`
	_, err := testDB.DB.Exec(query, time.Now(), networkID)
	require.NoError(t, err)
}

func countNetworksWithFQDN(t *testing.T, testDB *helper.TestDB, fqdn string) int {
	t.Helper()

	var count int
	query := `SELECT COUNT(*) FROM NETWORKS WHERE fqdn = ?`
	err := testDB.DB.QueryRow(query, fqdn).Scan(&count)
	require.NoError(t, err)

	return count
}

func getActiveNetworkFQDN(t *testing.T, testDB *helper.TestDB, networkID uint) string {
	t.Helper()

	var fqdn sql.NullString
	query := `SELECT fqdn FROM NETWORKS WHERE network_id = ? AND is_deleted = 0`
	err := testDB.DB.QueryRow(query, networkID).Scan(&fqdn)
	require.NoError(t, err)

	if fqdn.Valid {
		return fqdn.String
	}
	return ""
}

func getDeletedNetworkFQDN(t *testing.T, testDB *helper.TestDB, networkID uint) string {
	t.Helper()

	var fqdn sql.NullString
	query := `SELECT fqdn FROM NETWORKS WHERE network_id = ? AND is_deleted = 1`
	err := testDB.DB.QueryRow(query, networkID).Scan(&fqdn)
	require.NoError(t, err)

	if fqdn.Valid {
		return fqdn.String
	}
	return ""
}

func checkFQDNExistsForProject(t *testing.T, testDB *helper.TestDB, fqdn string, projectID uint) bool {
	t.Helper()

	// This query mimics CheckFQDNExistsForProject from networks.sql
	query := `
		SELECT COUNT(*) > 0 as fqdn_exists
		FROM NETWORKS n
		INNER JOIN CONTAINERS c ON n.container_id = c.container_id
		WHERE n.fqdn = ?
		  AND (
		    (c.project_id = ? AND n.is_deleted = 0)
		    OR (c.project_id != ?)
		  )
	`
	var exists bool
	err := testDB.DB.QueryRow(query, fqdn, projectID, projectID).Scan(&exists)
	require.NoError(t, err)

	return exists
}

func deleteProject(t *testing.T, testDB *helper.TestDB, projectID uint) {
	t.Helper()

	query := `DELETE FROM PROJECTS WHERE project_id = ?`
	_, err := testDB.DB.Exec(query, projectID)
	require.NoError(t, err)
}

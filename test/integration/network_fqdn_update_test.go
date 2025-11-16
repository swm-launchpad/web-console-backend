package integration

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

// TestNetworkFQDNUpdate_SoftDeleteAndCreate tests FQDN update behavior
func TestNetworkFQDNUpdate_SoftDeleteAndCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Should use soft delete + create pattern when FQDN changes", func(t *testing.T) {
		// Given: Create project, container, and network with FQDN
		projectID := createTestProjectForFQDN(t, testDB, "test-project-1")
		containerID := createTestContainerForFQDN(t, testDB, projectID, "container-1")
		oldFQDN := "old-domain.launchpad.kr"
		networkID := createTestNetworkForFQDN(t, testDB, containerID, oldFQDN, 8080)

		// When: Update FQDN to a new value
		newFQDN := "new-domain.launchpad.kr"
		updateNetworkFQDN(t, testDB, networkID, newFQDN)

		// Then: Old network should be soft deleted with FQDN preserved
		oldNetwork := getNetworkByID(t, testDB, networkID)
		assert.True(t, oldNetwork.IsDeleted, "Old network should be soft deleted")
		assert.Equal(t, oldFQDN, oldNetwork.FQDN, "Old FQDN should be preserved")

		// And: New network should exist with new FQDN and different network_id
		networks := getActiveNetworksByContainerID(t, testDB, containerID)
		require.Len(t, networks, 1, "Should have exactly one active network")
		assert.NotEqual(t, networkID, networks[0].NetworkID, "New network should have different network_id")
		assert.Equal(t, newFQDN, networks[0].FQDN, "New network should have new FQDN")
		assert.Equal(t, uint16(8080), networks[0].InternalPort, "Internal port should be preserved")
	})

	t.Run("Should use regular update when FQDN does not change", func(t *testing.T) {
		// Given: Create project, container, and network with FQDN
		projectID := createTestProjectForFQDN(t, testDB, "test-project-2")
		containerID := createTestContainerForFQDN(t, testDB, projectID, "container-2")
		fqdn := "same-domain.launchpad.kr"
		networkID := createTestNetworkForFQDN(t, testDB, containerID, fqdn, 8080)

		// When: Update internal port without changing FQDN
		updateNetworkPort(t, testDB, networkID, 9090)

		// Then: Network should not be soft deleted (still active with same network_id)
		network := getNetworkByID(t, testDB, networkID)
		assert.False(t, network.IsDeleted, "Network should not be soft deleted")
		assert.Equal(t, fqdn, network.FQDN, "FQDN should remain the same")
		assert.Equal(t, uint16(9090), network.InternalPort, "Internal port should be updated")

		// And: Should have only one active network
		networks := getActiveNetworksByContainerID(t, testDB, containerID)
		require.Len(t, networks, 1, "Should have exactly one active network")
		assert.Equal(t, networkID, networks[0].NetworkID, "Network ID should be unchanged")
	})

	t.Run("Should preserve old FQDN to prevent reuse by different project", func(t *testing.T) {
		// Given: Project A has a network with FQDN
		projectAID := createTestProjectForFQDN(t, testDB, "project-a")
		containerAID := createTestContainerForFQDN(t, testDB, projectAID, "container-a")
		oldFQDN := "changed-domain.launchpad.kr"
		networkID := createTestNetworkForFQDN(t, testDB, containerAID, oldFQDN, 8080)

		// When: Project A changes FQDN
		newFQDN := "new-changed-domain.launchpad.kr"
		updateNetworkFQDN(t, testDB, networkID, newFQDN)

		// Then: Old FQDN should still be blocked from other projects
		projectBID := createTestProjectForFQDN(t, testDB, "project-b")
		exists := checkFQDNExistsForProject(t, testDB, oldFQDN, projectBID)
		assert.True(t, exists, "Old FQDN should still be blocked for different project")

		// And: Same project can reuse old FQDN after soft delete
		exists = checkFQDNExistsForProject(t, testDB, oldFQDN, projectAID)
		assert.False(t, exists, "Same project should be able to reuse old FQDN")
	})
}

// Helper types and functions

type networkInfo struct {
	NetworkID    uint
	ContainerID  uint
	FQDN         string
	InternalPort uint16
	IsDeleted    bool
}

func updateNetworkFQDN(t *testing.T, testDB *helper.TestDB, networkID uint, newFQDN string) {
	t.Helper()

	query := `
		UPDATE NETWORKS SET
			fqdn = ?,
			updated_at = NOW()
		WHERE network_id = ?
	`
	_, err := testDB.DB.Exec(query, newFQDN, networkID)
	require.NoError(t, err)
}

func updateNetworkPort(t *testing.T, testDB *helper.TestDB, networkID uint, newPort uint16) {
	t.Helper()

	query := `
		UPDATE NETWORKS SET
			internal_port = ?,
			updated_at = NOW()
		WHERE network_id = ?
	`
	_, err := testDB.DB.Exec(query, newPort, networkID)
	require.NoError(t, err)
}

func getNetworkByID(t *testing.T, testDB *helper.TestDB, networkID uint) networkInfo {
	t.Helper()

	var info networkInfo
	query := `
		SELECT network_id, container_id, fqdn, internal_port, is_deleted
		FROM NETWORKS
		WHERE network_id = ?
	`
	err := testDB.DB.QueryRow(query, networkID).Scan(
		&info.NetworkID,
		&info.ContainerID,
		&info.FQDN,
		&info.InternalPort,
		&info.IsDeleted,
	)
	require.NoError(t, err)

	return info
}

func getActiveNetworksByContainerID(t *testing.T, testDB *helper.TestDB, containerID uint) []networkInfo {
	t.Helper()

	query := `
		SELECT network_id, container_id, fqdn, internal_port, is_deleted
		FROM NETWORKS
		WHERE container_id = ? AND is_deleted = 0
	`
	rows, err := testDB.DB.Query(query, containerID)
	require.NoError(t, err)
	defer func() {
		_ = rows.Close()
	}()

	var networks []networkInfo
	for rows.Next() {
		var info networkInfo
		var fqdn sql.NullString
		err := rows.Scan(
			&info.NetworkID,
			&info.ContainerID,
			&fqdn,
			&info.InternalPort,
			&info.IsDeleted,
		)
		require.NoError(t, err)

		if fqdn.Valid {
			info.FQDN = fqdn.String
		}
		networks = append(networks, info)
	}
	require.NoError(t, rows.Err())

	return networks
}

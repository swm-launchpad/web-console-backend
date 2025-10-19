package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	containerApp "github.com/swm-launchpad/web-console-backend/internal/container/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

// setupTestUserAndProject creates a test user and project in the database
func setupTestUserAndProject(t *testing.T, testDB *helper.TestDB, userID uint, projectID uint) {
	t.Helper()

	// Create user
	username := "testuser"
	if userID != 100 {
		username = "testuser" + string(rune(userID))
	}
	email := username + "@example.com"

	_, err := testDB.DB.Exec("INSERT INTO USERS (user_id, username, email, password_hash, status) VALUES (?, ?, ?, ?, ?)",
		userID, username, email, "hashedpassword", "active")
	require.NoError(t, err)

	// Create project
	_, err = testDB.DB.Exec("INSERT INTO PROJECTS (project_id, name, slug, status, cpu_limit, memory_limit) VALUES (?, ?, ?, ?, ?, ?)",
		projectID, "Test Project", "test-project", "active", 10000, 10240)
	require.NoError(t, err)

	// Link user to project as owner
	_, err = testDB.DB.Exec("INSERT INTO PROJECT_USER (project_id, user_id, role) VALUES (?, ?, ?)",
		projectID, userID, "owner")
	require.NoError(t, err)
}

// setupTestUser creates only a user in the database
func setupTestUser(t *testing.T, testDB *helper.TestDB, userID uint, username string) {
	t.Helper()

	email := username + "@example.com"
	_, err := testDB.DB.Exec("INSERT INTO USERS (user_id, username, email, password_hash, status) VALUES (?, ?, ?, ?, ?)",
		userID, username, email, "hashedpassword", "active")
	require.NoError(t, err)
}

func TestGitHubInstallationSecurity_CreateContainer_WithValidInstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	testServer := helper.SetupTestServer(t)
	defer testServer.DB.Cleanup()

	ctx := context.Background()

	// Create test user and project
	userID := uint(100)
	projectID := uint(10)
	installationID := int64(12345)

	setupTestUserAndProject(t, testDB, userID, projectID)

	// Create GitHub installation for the user
	installation := &model.GitHubInstallation{
		InstallationID: installationID,
		UserID:         userID,
		AccountLogin:   "testuser",
		AccountType:    model.AccountTypeUser,
		Status:         model.InstallationStatusActive,
	}

	installationRepo := helper.GetGitHubInstallationRepository(testDB)
	err := installationRepo.Create(ctx, installation)
	require.NoError(t, err)

	// Attempt to create container with own installation ID
	createUseCase := helper.GetCreateContainerUseCase(testServer)
	input := containerApp.CreateContainerInput{
		ProjectID:            projectID,
		UserID:               userID,
		Name:                 "test-container",
		GitURL:               "https://github.com/testuser/private-repo",
		GitBranch:            "main",
		GitHubInstallationID: &installationID,
		CPULimit:             1000,
		MemoryLimit:          512,
	}

	_, err = createUseCase.Execute(ctx, input)

	// Should succeed - user owns the installation
	assert.NoError(t, err)
}

func TestGitHubInstallationSecurity_CreateContainer_WithUnauthorizedInstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	testServer := helper.SetupTestServer(t)
	defer testServer.DB.Cleanup()

	ctx := context.Background()

	// Create two users
	userA := uint(100)
	userB := uint(200)
	projectID := uint(10)
	installationIDOfUserB := int64(67890)

	setupTestUserAndProject(t, testDB, userA, projectID)
	setupTestUser(t, testDB, userB, "userB")

	// Create GitHub installation for User B
	installation := &model.GitHubInstallation{
		InstallationID: installationIDOfUserB,
		UserID:         userB,
		AccountLogin:   "userB",
		AccountType:    model.AccountTypeUser,
		Status:         model.InstallationStatusActive,
	}

	installationRepo := helper.GetGitHubInstallationRepository(testDB)
	err := installationRepo.Create(ctx, installation)
	require.NoError(t, err)

	// User A attempts to create container with User B's installation ID
	createUseCase := helper.GetCreateContainerUseCase(testServer)
	input := containerApp.CreateContainerInput{
		ProjectID:            projectID,
		UserID:               userA,
		Name:                 "malicious-container",
		GitURL:               "https://github.com/userB/private-repo",
		GitBranch:            "main",
		GitHubInstallationID: &installationIDOfUserB, // Unauthorized!
		CPULimit:             1000,
		MemoryLimit:          512,
	}

	_, err = createUseCase.Execute(ctx, input)

	// Should fail - User A doesn't own this installation
	assert.Error(t, err)
	assert.ErrorIs(t, err, usererrors.ErrInstallationUnauthorized)
}

func TestGitHubInstallationSecurity_CreateContainer_WithRevokedInstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	testServer := helper.SetupTestServer(t)
	defer testServer.DB.Cleanup()

	ctx := context.Background()

	userID := uint(100)
	projectID := uint(10)
	installationID := int64(12345)

	setupTestUserAndProject(t, testDB, userID, projectID)

	// Create GitHub installation
	installation := &model.GitHubInstallation{
		InstallationID: installationID,
		UserID:         userID,
		AccountLogin:   "testuser",
		AccountType:    model.AccountTypeUser,
		Status:         model.InstallationStatusActive,
	}

	installationRepo := helper.GetGitHubInstallationRepository(testDB)
	err := installationRepo.Create(ctx, installation)
	require.NoError(t, err)

	// Revoke the installation
	err = installationRepo.MarkAsRevoked(ctx, installationID)
	require.NoError(t, err)

	// Attempt to create container with revoked installation
	createUseCase := helper.GetCreateContainerUseCase(testServer)
	input := containerApp.CreateContainerInput{
		ProjectID:            projectID,
		UserID:               userID,
		Name:                 "test-container",
		GitURL:               "https://github.com/testuser/private-repo",
		GitBranch:            "main",
		GitHubInstallationID: &installationID,
		CPULimit:             1000,
		MemoryLimit:          512,
	}

	_, err = createUseCase.Execute(ctx, input)

	// Should fail - Installation is revoked
	assert.Error(t, err)
	assert.ErrorIs(t, err, usererrors.ErrInstallationUnauthorized)
}

func TestGitHubInstallationSecurity_CreateContainer_WithNilInstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	testServer := helper.SetupTestServer(t)
	defer testServer.DB.Cleanup()

	ctx := context.Background()

	userID := uint(100)
	projectID := uint(10)

	setupTestUserAndProject(t, testDB, userID, projectID)

	// Attempt to create container without installation ID (public repo)
	createUseCase := helper.GetCreateContainerUseCase(testServer)
	input := containerApp.CreateContainerInput{
		ProjectID:            projectID,
		UserID:               userID,
		Name:                 "public-repo-container",
		GitURL:               "https://github.com/public/repo",
		GitBranch:            "main",
		GitHubInstallationID: nil, // No installation needed for public repos
		CPULimit:             1000,
		MemoryLimit:          512,
	}

	_, err := createUseCase.Execute(ctx, input)

	// Should succeed - No validation needed for nil installation
	assert.NoError(t, err)
}

func TestGitHubInstallationSecurity_UpdateContainer_SecurityValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	testServer := helper.SetupTestServer(t)
	defer testServer.DB.Cleanup()

	ctx := context.Background()

	userA := uint(100)
	userB := uint(200)
	containerID := uint(1)
	projectID := uint(10)
	installationIDOfUserB := int64(67890)

	setupTestUserAndProject(t, testDB, userA, projectID)
	setupTestUser(t, testDB, userB, "userB")

	// Create GitHub installation for User B
	installation := &model.GitHubInstallation{
		InstallationID: installationIDOfUserB,
		UserID:         userB,
		AccountLogin:   "userB",
		AccountType:    model.AccountTypeUser,
		Status:         model.InstallationStatusActive,
	}

	installationRepo := helper.GetGitHubInstallationRepository(testDB)
	err := installationRepo.Create(ctx, installation)
	require.NoError(t, err)

	// User A attempts to update container with User B's installation ID
	updateUseCase := helper.GetUpdateContainerUseCase(testServer)
	input := containerApp.UpdateContainerInput{
		ContainerID:                containerID,
		UserID:                     userA,
		GitHubInstallationID:       &installationIDOfUserB, // Unauthorized!
		UpdateGitHubInstallationID: true,
	}

	_, err = updateUseCase.Execute(ctx, input)

	// Should fail - User A doesn't own this installation
	assert.Error(t, err)
	assert.ErrorIs(t, err, usererrors.ErrInstallationUnauthorized)
}

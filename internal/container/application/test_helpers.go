package application

import (
	"fmt"
	"time"

	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

// Helper functions for tests

// createMockContainer creates a basic empty container for testing (no env vars, networks, secrets)
func createMockContainer(containerID, projectID uint) *model.Container {
	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	gitConfig, _ := value.NewGitConfig(
		"https://github.com/test/repo",
		"main",
		nil,
	)

	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := model.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"Test Container",
		slug,
		nil,
		nil,
		nil,
		gitConfig,
		nil,
		nil,
		true, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	// Container is empty by default (no env vars, networks, or secrets)
	return c
}

// createMockDeletedContainer creates a soft-deleted container for testing
func createMockDeletedContainer(containerID, projectID uint) *model.Container {
	slug, _ := value.NewContainerSlug("c2025011812000087654321")
	gitConfig, _ := value.NewGitConfig(
		"https://github.com/test/repo",
		"main",
		nil,
	)

	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := model.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"deleted-container",
		slug,
		nil,
		nil,
		nil,
		gitConfig,
		nil,
		nil,
		true, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		true, // is_deleted = true
		nil,
		time.Now(),
		time.Now(),
	)

	return c
}

// createMockEnvVar creates a mock environment variable
func createMockEnvVar(envVarID, containerID uint, key, value string) *model.EnvVar {
	envVar := model.ReconstructEnvVar(envVarID, containerID, key, value, time.Now(), time.Now())
	return envVar
}

// createMockNetwork creates a mock network
func createMockNetwork(networkID, containerID uint, containerPort uint16, protocol string) *model.Network {
	networkType, _ := value.NewNetworkType(protocol)
	hostPort := containerPort + 10000
	network := model.ReconstructNetwork(
		networkID,
		containerID,
		&containerPort,
		&hostPort,
		networkType,
		nil,
		nil,
		nil, // tektonEventID
		nil, // expiresAt
		time.Now(),
		time.Now(),
	)
	return network
}

// createMockNetworkWithFQDN creates a mock network with FQDN
func createMockNetworkWithFQDN(networkID, containerID uint, internalPort, externalPort *uint16, networkType value.NetworkType, fqdn *string) *model.Network {
	network := model.ReconstructNetwork(
		networkID,
		containerID,
		internalPort,
		externalPort,
		networkType,
		nil,
		fqdn,
		nil, // tektonEventID
		nil, // expiresAt
		time.Now(),
		time.Now(),
	)
	return network
}

// createMockSecret creates a mock secret
func createMockSecret(secretID, containerID uint, key, value string) *model.Secret {
	secret := model.ReconstructSecret(secretID, containerID, key, value, time.Now(), time.Now())
	return secret
}

// createMockContainerWithMaxEnvVars creates a container with maximum number of env vars
func createMockContainerWithMaxEnvVars(containerID, projectID uint) *model.Container {
	c := createMockContainer(containerID, projectID)
	// Add 100 env vars (max is 100 according to the domain constant MaxEnvVarsPerContainer)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("VAR_%d", i)
		_ = c.AddEnvVarDirect(createMockEnvVar(uint(i+1), containerID, key, "value"))
	}
	return c
}

// createMockContainerWithMaxNetworks creates a container with maximum number of networks
func createMockContainerWithMaxNetworks(containerID, projectID uint) *model.Container {
	c := createMockContainer(containerID, projectID)
	// Add 20 networks (max is 20 according to the domain)
	for i := 0; i < 20; i++ {
		_ = c.AddNetworkDirect(createMockNetwork(uint(i+1), containerID, uint16(8000+i), "tcp"))
	}
	return c
}

// createMockContainerWithOptionalFields creates a container with optional fields
func createMockContainerWithOptionalFields(
	containerID, projectID uint,
	fqdn, gitDir *string,
	stableWindow *uint32,
	templateConfig map[string]interface{},
) *model.Container {
	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	gitConfig, _ := value.NewGitConfig(
		"https://github.com/test/repo",
		"main",
		gitDir,
	)

	cpuLimit := uint32(1000)
	memoryLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memoryLimit)

	c := model.ReconstructContainer(
		containerID,
		projectID,
		nil,
		"Test Container",
		slug,
		stableWindow,
		templateConfig,
		nil,
		gitConfig,
		nil,
		nil,
		true, // needsBuild
		resourceLimits,
		nil,
		nil,
		nil,
		false,
		nil,
		time.Now(),
		time.Now(),
	)

	return c
}

// createMockContainerWithEnvVarsAndNetworks creates a container with env vars and networks for testing
func createMockContainerWithEnvVarsAndNetworks(containerID, projectID uint) *model.Container {
	c := createMockContainer(containerID, projectID)

	// Add some env vars
	_ = c.AddEnvVarDirect(createMockEnvVar(1, containerID, "DATABASE_URL", "postgres://..."))
	_ = c.AddEnvVarDirect(createMockEnvVar(2, containerID, "API_KEY", "secret123"))

	// Add some networks
	_ = c.AddNetworkDirect(createMockNetwork(1, containerID, 8080, "tcp"))
	_ = c.AddNetworkDirect(createMockNetwork(2, containerID, 3000, "tcp"))

	return c
}

// createMockContainerWithSecretsAndNetworks creates a container with secrets and networks for testing
func createMockContainerWithSecretsAndNetworks(containerID, projectID uint) *model.Container {
	c := createMockContainer(containerID, projectID)

	// Add some secrets
	_ = c.AddSecretDirect(createMockSecret(1, containerID, "DATABASE_PASSWORD", "super_secret"))
	_ = c.AddSecretDirect(createMockSecret(2, containerID, "API_SECRET_KEY", "secret_key_123"))

	// Add some networks
	_ = c.AddNetworkDirect(createMockNetwork(1, containerID, 8080, "tcp"))
	_ = c.AddNetworkDirect(createMockNetwork(2, containerID, 3000, "tcp"))

	return c
}

// createMockContainerWithMaxSecrets creates a container with maximum number of secrets
func createMockContainerWithMaxSecrets(containerID, projectID uint) *model.Container {
	c := createMockContainer(containerID, projectID)
	// Add 100 secrets (max is 100 according to the domain constant MaxSecretsPerContainer)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("SECRET_%d", i)
		_ = c.AddSecretDirect(createMockSecret(uint(i+1), containerID, key, "secret_value"))
	}
	return c
}

// createMockContainerWithMounts creates a container with 2 mounts for testing
func createMockContainerWithMounts(containerID, projectID uint) *model.Container {
	c := createMockContainer(containerID, projectID)

	// Add 2 mounts
	mount1, _ := model.NewMount(containerID, uint(1000), "/app/data1")
	mount2, _ := model.NewMount(containerID, uint(1001), "/app/data2")

	_ = c.AddMountDirect(mount1)
	_ = c.AddMountDirect(mount2)

	return c
}

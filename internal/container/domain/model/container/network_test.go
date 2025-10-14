package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

func TestNewNetwork_Success(t *testing.T) {
	containerID := uint(1)
	internalPort := uint16(8080)
	externalPort := uint16(80)
	networkType, _ := value.NewNetworkType("tcp")
	externalIP := "0.0.0.0"
	fqdn := "example.com"

	network, err := NewNetwork(containerID, &internalPort, &externalPort, networkType, &externalIP, &fqdn)

	require.NoError(t, err)
	assert.NotNil(t, network)
	assert.Equal(t, containerID, network.ContainerID())
	assert.Equal(t, &internalPort, network.InternalPort())
	assert.Equal(t, &externalPort, network.ExternalPort())
	assert.Equal(t, networkType, network.NetworkType())
	assert.Equal(t, &externalIP, network.ExternalIP())
	assert.Equal(t, &fqdn, network.FQDN())
	assert.NotZero(t, network.CreatedAt())
	assert.True(t, network.UpdatedAt().IsZero())
}

func TestNewNetwork_OptionalFields(t *testing.T) {
	t.Run("Only internal port", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("tcp")

		network, err := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, network)
		assert.Equal(t, &internalPort, network.InternalPort())
		assert.Nil(t, network.ExternalPort())
		assert.Nil(t, network.ExternalIP())
		assert.Nil(t, network.FQDN())
	})

	t.Run("Both ports set", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		externalPort := uint16(80)
		networkType, _ := value.NewNetworkType("http")

		network, err := NewNetwork(containerID, &internalPort, &externalPort, networkType, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, network)
		assert.Equal(t, &internalPort, network.InternalPort())
		assert.Equal(t, &externalPort, network.ExternalPort())
		assert.Nil(t, network.ExternalIP())
		assert.Nil(t, network.FQDN())
	})

	t.Run("With FQDN", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("http")
		fqdn := "app.example.com"

		network, err := NewNetwork(containerID, &internalPort, nil, networkType, nil, &fqdn)

		require.NoError(t, err)
		assert.NotNil(t, network)
		assert.Equal(t, &fqdn, network.FQDN())
	})
}

func TestNewNetwork_InvalidContainerID(t *testing.T) {
	internalPort := uint16(8080)
	networkType, _ := value.NewNetworkType("tcp")

	network, err := NewNetwork(0, &internalPort, nil, networkType, nil, nil)

	assert.ErrorIs(t, err, containererrors.ErrInvalidContainerID)
	assert.Nil(t, network)
}

func TestNewNetwork_MissingInternalPort(t *testing.T) {
	containerID := uint(1)
	networkType, _ := value.NewNetworkType("tcp")

	network, err := NewNetwork(containerID, nil, nil, networkType, nil, nil)

	assert.ErrorIs(t, err, containererrors.ErrInternalPortRequired)
	assert.Nil(t, network)
}

func TestNewNetwork_InvalidPortRange(t *testing.T) {
	containerID := uint(1)
	networkType, _ := value.NewNetworkType("tcp")

	t.Run("Internal port 0", func(t *testing.T) {
		invalidPort := uint16(0)
		network, err := NewNetwork(containerID, &invalidPort, nil, networkType, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrPortOutOfRange)
		assert.Nil(t, network)
	})

	t.Run("External port 0", func(t *testing.T) {
		internalPort := uint16(8080)
		invalidPort := uint16(0)
		network, err := NewNetwork(containerID, &internalPort, &invalidPort, networkType, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrPortOutOfRange)
		assert.Nil(t, network)
	})

	t.Run("Valid boundary ports", func(t *testing.T) {
		internalPort := uint16(1)
		externalPort := uint16(65535)
		network, err := NewNetwork(containerID, &internalPort, &externalPort, networkType, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, network)
	})
}

func TestNewNetwork_InvalidExternalIP(t *testing.T) {
	containerID := uint(1)
	internalPort := uint16(8080)
	networkType, _ := value.NewNetworkType("tcp")

	t.Run("Empty IP", func(t *testing.T) {
		emptyIP := ""
		network, err := NewNetwork(containerID, &internalPort, nil, networkType, &emptyIP, nil)
		assert.ErrorIs(t, err, containererrors.ErrInvalidExternalIP)
		assert.Nil(t, network)
	})

	t.Run("IP too long", func(t *testing.T) {
		// Create a 46-character IP string (exceeds 45 char limit)
		longIP := "1234567890123456789012345678901234567890123456"
		network, err := NewNetwork(containerID, &internalPort, nil, networkType, &longIP, nil)
		assert.ErrorIs(t, err, containererrors.ErrExternalIPTooLong)
		assert.Nil(t, network)
	})
}

func TestNewNetwork_InvalidFQDN(t *testing.T) {
	containerID := uint(1)
	internalPort := uint16(8080)
	networkType, _ := value.NewNetworkType("tcp")

	t.Run("Empty FQDN", func(t *testing.T) {
		emptyFQDN := ""
		network, err := NewNetwork(containerID, &internalPort, nil, networkType, nil, &emptyFQDN)
		assert.ErrorIs(t, err, containererrors.ErrInvalidFQDN)
		assert.Nil(t, network)
	})

	t.Run("FQDN too long", func(t *testing.T) {
		// Create a 256-character FQDN string (exceeds 255 char limit)
		longFQDN := string(make([]byte, 256))
		for i := range longFQDN {
			longFQDN = string(append([]byte(longFQDN[:i]), 'a'))
		}
		network, err := NewNetwork(containerID, &internalPort, nil, networkType, nil, &longFQDN)
		assert.ErrorIs(t, err, containererrors.ErrFQDNTooLong)
		assert.Nil(t, network)
	})
}

func TestNetwork_UpdateExternalPort(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		oldPort := uint16(80)
		networkType, _ := value.NewNetworkType("tcp")

		network, _ := NewNetwork(containerID, &internalPort, &oldPort, networkType, nil, nil)
		newPort := uint16(8000)

		err := network.UpdateExternalPort(&newPort)

		require.NoError(t, err)
		assert.Equal(t, &newPort, network.ExternalPort())
		assert.False(t, network.UpdatedAt().IsZero())
	})

	t.Run("Remove external port (set to nil)", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		oldPort := uint16(80)
		networkType, _ := value.NewNetworkType("tcp")

		network, _ := NewNetwork(containerID, &internalPort, &oldPort, networkType, nil, nil)

		err := network.UpdateExternalPort(nil)

		require.NoError(t, err)
		assert.Nil(t, network.ExternalPort())
	})

	t.Run("Invalid port (0)", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("tcp")

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)
		invalidPort := uint16(0)

		err := network.UpdateExternalPort(&invalidPort)

		assert.ErrorIs(t, err, containererrors.ErrPortOutOfRange)
	})
}

func TestNetwork_UpdateExternalIP(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("tcp")

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)
		newIP := "192.168.1.1"

		err := network.UpdateExternalIP(&newIP)

		require.NoError(t, err)
		assert.Equal(t, &newIP, network.ExternalIP())
		assert.False(t, network.UpdatedAt().IsZero())
	})

	t.Run("Remove external IP (set to nil)", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("tcp")
		oldIP := "0.0.0.0"

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, &oldIP, nil)

		err := network.UpdateExternalIP(nil)

		require.NoError(t, err)
		assert.Nil(t, network.ExternalIP())
	})

	t.Run("Empty IP", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("tcp")

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)
		emptyIP := ""

		err := network.UpdateExternalIP(&emptyIP)

		assert.ErrorIs(t, err, containererrors.ErrInvalidExternalIP)
	})
}

func TestNetwork_SetFQDN(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("http")

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)
		fqdn := "app.example.com"

		err := network.SetFQDN(&fqdn)

		require.NoError(t, err)
		assert.Equal(t, &fqdn, network.FQDN())
		assert.False(t, network.UpdatedAt().IsZero())
	})

	t.Run("Update existing FQDN", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("http")
		oldFQDN := "old.example.com"

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, &oldFQDN)
		newFQDN := "new.example.com"

		err := network.SetFQDN(&newFQDN)

		require.NoError(t, err)
		assert.Equal(t, &newFQDN, network.FQDN())
	})

	t.Run("Remove FQDN (set to nil)", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("http")
		oldFQDN := "example.com"

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, &oldFQDN)

		err := network.SetFQDN(nil)

		require.NoError(t, err)
		assert.Nil(t, network.FQDN())
	})

	t.Run("Empty FQDN", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("http")

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)
		emptyFQDN := ""

		err := network.SetFQDN(&emptyFQDN)

		assert.ErrorIs(t, err, containererrors.ErrInvalidFQDN)
	})

	t.Run("FQDN too long", func(t *testing.T) {
		containerID := uint(1)
		internalPort := uint16(8080)
		networkType, _ := value.NewNetworkType("http")

		network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)
		// Create a 256-character FQDN string (exceeds 255 char limit)
		longFQDN := ""
		for i := 0; i < 256; i++ {
			longFQDN += "a"
		}

		err := network.SetFQDN(&longFQDN)

		assert.ErrorIs(t, err, containererrors.ErrFQDNTooLong)
	})
}

func TestNetwork_SetNetworkID(t *testing.T) {
	containerID := uint(1)
	internalPort := uint16(8080)
	networkType, _ := value.NewNetworkType("tcp")

	network, _ := NewNetwork(containerID, &internalPort, nil, networkType, nil, nil)

	assert.Equal(t, uint(0), network.NetworkID())

	network.SetNetworkID(999)
	assert.Equal(t, uint(999), network.NetworkID())
}

func TestReconstructNetwork(t *testing.T) {
	networkID := uint(100)
	containerID := uint(1)
	internalPort := uint16(8080)
	externalPort := uint16(80)
	networkType, _ := value.NewNetworkType("tcp")
	externalIP := "0.0.0.0"
	fqdn := "test.example.com"

	network := ReconstructNetwork(networkID, containerID, &internalPort, &externalPort, networkType, &externalIP, &fqdn, time.Now(), time.Now())

	assert.NotNil(t, network)
	assert.Equal(t, networkID, network.NetworkID())
	assert.Equal(t, containerID, network.ContainerID())
	assert.Equal(t, &internalPort, network.InternalPort())
	assert.Equal(t, &externalPort, network.ExternalPort())
	assert.Equal(t, networkType, network.NetworkType())
	assert.Equal(t, &externalIP, network.ExternalIP())
	assert.Equal(t, &fqdn, network.FQDN())
}

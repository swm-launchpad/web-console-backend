package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewNetworkType_ValidTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected NetworkType
	}{
		{input: "tcp", expected: NetworkTypeTCP},
		{input: "TCP", expected: NetworkTypeTCP},
		{input: "udp", expected: NetworkTypeUDP},
		{input: "UDP", expected: NetworkTypeUDP},
		{input: "http", expected: NetworkTypeHTTP},
		{input: "HTTP", expected: NetworkTypeHTTP},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NewNetworkType(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewNetworkType_Invalid(t *testing.T) {
	invalidTypes := []string{
		"ftp",
		"ssh",
		"https",
		"",
		"tcpudp",
	}

	for _, typeStr := range invalidTypes {
		t.Run(typeStr, func(t *testing.T) {
			_, err := NewNetworkType(typeStr)
			assert.ErrorIs(t, err, containererrors.ErrInvalidNetworkType)
		})
	}
}

func TestNetworkType_String(t *testing.T) {
	assert.Equal(t, "tcp", NetworkTypeTCP.String())
	assert.Equal(t, "udp", NetworkTypeUDP.String())
	assert.Equal(t, "http", NetworkTypeHTTP.String())
}

func TestNetworkType_Is(t *testing.T) {
	tcp, _ := NewNetworkType("tcp")
	assert.True(t, tcp.IsTCP())
	assert.False(t, tcp.IsUDP())
	assert.False(t, tcp.IsHTTP())

	udp, _ := NewNetworkType("udp")
	assert.False(t, udp.IsTCP())
	assert.True(t, udp.IsUDP())
	assert.False(t, udp.IsHTTP())

	http, _ := NewNetworkType("http")
	assert.False(t, http.IsTCP())
	assert.False(t, http.IsUDP())
	assert.True(t, http.IsHTTP())
}

func TestNetworkType_Equals(t *testing.T) {
	tcp1, _ := NewNetworkType("tcp")
	tcp2, _ := NewNetworkType("TCP")
	udp, _ := NewNetworkType("udp")

	assert.True(t, tcp1.Equals(tcp2))
	assert.False(t, tcp1.Equals(udp))
}

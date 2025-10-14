package value

import (
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// NetworkType represents the protocol type for network connections
// It is a value object that ensures only valid network types are used
type NetworkType string

const (
	NetworkTypeTCP  NetworkType = "tcp"
	NetworkTypeUDP  NetworkType = "udp"
	NetworkTypeHTTP NetworkType = "http"
)

// NewNetworkType creates a new NetworkType with validation
func NewNetworkType(typeStr string) (NetworkType, error) {
	switch typeStr {
	case "tcp", "TCP":
		return NetworkTypeTCP, nil
	case "udp", "UDP":
		return NetworkTypeUDP, nil
	case "http", "HTTP":
		return NetworkTypeHTTP, nil
	default:
		return "", containererrors.ErrInvalidNetworkType
	}
}

// String returns the string representation of the network type
func (n NetworkType) String() string {
	return string(n)
}

// Equals checks if two NetworkTypes are equal
func (n NetworkType) Equals(other NetworkType) bool {
	return n == other
}

// IsTCP returns true if the type is TCP
func (n NetworkType) IsTCP() bool {
	return n == NetworkTypeTCP
}

// IsUDP returns true if the type is UDP
func (n NetworkType) IsUDP() bool {
	return n == NetworkTypeUDP
}

// IsHTTP returns true if the type is HTTP
func (n NetworkType) IsHTTP() bool {
	return n == NetworkTypeHTTP
}

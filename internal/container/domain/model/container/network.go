package model

import (
	"strings"
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

// Network represents a network port mapping within the Container aggregate
// This is an entity within the Container aggregate
type Network struct {
	networkID    uint
	containerID  uint
	externalIP   *string
	externalPort *uint16
	internalPort *uint16
	fqdn         *string
	networkType  value.NetworkType
	createdAt    time.Time
	updatedAt    time.Time
}

const (
	MinPort = 1
	MaxPort = 65535
)

// Reserved subdomains that cannot be used by users
var reservedSubdomains = map[string]bool{
	"tekton":       true,
	"tekton-api":   true,
	"kube-api":     true,
	"registry":     true,
	"grafana":      true,
	"vm":           true,
	"vmalert":      true,
	"alertmanager": true,
	"loki":         true,
	"api":          true,
	"www":          true,
}

const (
	minSubdomainLength = 4 // Subdomains must be at least 4 characters
	launchpadDomain    = ".launchpad.kr"
)

// extractSubdomain extracts the subdomain from a FQDN
// e.g., "myapp.launchpad.kr" -> "myapp"
func extractSubdomain(fqdn string) string {
	if !strings.HasSuffix(fqdn, launchpadDomain) {
		return fqdn
	}
	return strings.TrimSuffix(fqdn, launchpadDomain)
}

// isReservedSubdomain checks if a subdomain is reserved
func isReservedSubdomain(subdomain string) bool {
	return reservedSubdomains[strings.ToLower(subdomain)]
}

// validateFQDN validates the FQDN against business rules
func validateFQDN(fqdn string) error {
	if fqdn == "" {
		return containererrors.ErrInvalidFQDN
	}
	if len(fqdn) > 255 {
		return containererrors.ErrFQDNTooLong
	}

	// Extract subdomain and validate
	subdomain := extractSubdomain(fqdn)

	// Check if subdomain is reserved first (more specific error message)
	if isReservedSubdomain(subdomain) {
		return containererrors.ErrReservedFQDN
	}

	// Check minimum length (must be at least 4 characters)
	if len(subdomain) < minSubdomainLength {
		return containererrors.ErrFQDNTooShort
	}

	return nil
}

// NewNetwork creates a new Network entity with validation
func NewNetwork(containerID uint, internalPort, externalPort *uint16, networkType value.NetworkType, externalIP *string, fqdn *string) (*Network, error) {
	if containerID == 0 {
		return nil, containererrors.ErrInvalidContainerID
	}

	// Internal port is required
	if internalPort == nil {
		return nil, containererrors.ErrInternalPortRequired
	}

	// Validate internal port
	if *internalPort < MinPort || *internalPort > MaxPort {
		return nil, containererrors.ErrPortOutOfRange
	}

	// Validate external port if provided
	if externalPort != nil {
		if *externalPort < MinPort || *externalPort > MaxPort {
			return nil, containererrors.ErrPortOutOfRange
		}
	}

	// Validate external IP if provided
	if externalIP != nil {
		if *externalIP == "" {
			return nil, containererrors.ErrInvalidExternalIP
		}
		if len(*externalIP) > 45 {
			return nil, containererrors.ErrExternalIPTooLong
		}
	}

	// Validate FQDN if provided
	if fqdn != nil {
		if err := validateFQDN(*fqdn); err != nil {
			return nil, err
		}
	}

	// Validate network type
	if networkType == "" {
		return nil, containererrors.ErrInvalidNetworkType
	}

	return &Network{
		containerID:  containerID,
		externalIP:   externalIP,
		externalPort: externalPort,
		internalPort: internalPort,
		fqdn:         fqdn,
		networkType:  networkType,
		createdAt:    time.Now(),
		updatedAt:    time.Time{}, // Zero time for new networks (NULL in database)
	}, nil
}

// NetworkID returns the network ID
func (n *Network) NetworkID() uint {
	return n.networkID
}

// ContainerID returns the container ID
func (n *Network) ContainerID() uint {
	return n.containerID
}

// ExternalIP returns the external IP address (may be nil)
func (n *Network) ExternalIP() *string {
	return n.externalIP
}

// ExternalPort returns the external port (may be nil)
func (n *Network) ExternalPort() *uint16 {
	return n.externalPort
}

// InternalPort returns the internal port (may be nil)
func (n *Network) InternalPort() *uint16 {
	return n.internalPort
}

// NetworkType returns the network type (tcp, udp, http)
func (n *Network) NetworkType() value.NetworkType {
	return n.networkType
}

// FQDN returns the fully qualified domain name (may be nil)
func (n *Network) FQDN() *string {
	return n.fqdn
}

// CreatedAt returns the creation timestamp
func (n *Network) CreatedAt() time.Time {
	return n.createdAt
}

// UpdatedAt returns the last update timestamp
func (n *Network) UpdatedAt() time.Time {
	return n.updatedAt
}

// SetNetworkID sets the network ID (used by repository after persistence)
func (n *Network) SetNetworkID(id uint) {
	n.networkID = id
}

// SetExternalIP sets the external IP address
func (n *Network) SetExternalIP(ip *string) {
	n.externalIP = ip
	n.updatedAt = time.Now()
}

// SetFQDN sets the fully qualified domain name
func (n *Network) SetFQDN(fqdn *string) error {
	// Validate FQDN if provided
	if fqdn != nil {
		if err := validateFQDN(*fqdn); err != nil {
			return err
		}
	}
	n.fqdn = fqdn
	n.updatedAt = time.Now()
	return nil
}

// SetInternalPort sets the internal port with validation
func (n *Network) SetInternalPort(port *uint16) error {
	if port == nil {
		return containererrors.ErrInvalidPort
	}
	if *port < MinPort || *port > MaxPort {
		return containererrors.ErrInvalidPort
	}
	n.internalPort = port
	n.updatedAt = time.Now()
	return nil
}

// SetNetworkType sets the network type
func (n *Network) SetNetworkType(netType value.NetworkType) {
	n.networkType = netType
	n.updatedAt = time.Now()
}

// Equals checks if two Networks have the same internal port
func (n *Network) Equals(other *Network) bool {
	if other == nil {
		return false
	}
	// Two networks are considered equal if they have the same internal port
	if (n.internalPort == nil) != (other.internalPort == nil) {
		return false
	}
	if n.internalPort != nil && *n.internalPort != *other.internalPort {
		return false
	}
	return true
}

// HasExternalPort checks if this network has an external port assigned
func (n *Network) HasExternalPort() bool {
	return n.externalPort != nil
}

// HasInternalPort checks if this network has an internal port assigned
func (n *Network) HasInternalPort() bool {
	return n.internalPort != nil
}

// UpdateExternalPort updates the external port
func (n *Network) UpdateExternalPort(newPort *uint16) error {
	// Validate port if provided (uint16 max is 65535, so only check lower bound)
	if newPort != nil && *newPort < 1 {
		return containererrors.ErrPortOutOfRange
	}

	n.externalPort = newPort
	n.updatedAt = time.Now()
	return nil
}

// UpdateExternalIP updates the external IP address
func (n *Network) UpdateExternalIP(newIP *string) error {
	// Validate IP if provided
	if newIP != nil {
		if *newIP == "" {
			return containererrors.ErrInvalidExternalIP
		}
		if len(*newIP) > 45 {
			return containererrors.ErrExternalIPTooLong
		}
	}

	n.externalIP = newIP
	n.updatedAt = time.Now()
	return nil
}

// ReconstructNetwork reconstructs a network from persistence
// This is used when loading a network from the database
func ReconstructNetwork(
	networkID uint,
	containerID uint,
	internalPort *uint16,
	externalPort *uint16,
	networkType value.NetworkType,
	externalIP *string,
	fqdn *string,
	createdAt time.Time,
	updatedAt time.Time,
) *Network {
	return &Network{
		networkID:    networkID,
		containerID:  containerID,
		internalPort: internalPort,
		externalPort: externalPort,
		networkType:  networkType,
		externalIP:   externalIP,
		fqdn:         fqdn,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

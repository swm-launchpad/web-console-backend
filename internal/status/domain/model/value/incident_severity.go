package value

import "fmt"

// IncidentSeverity represents the severity level of an incident
type IncidentSeverity string

const (
	SeverityMinor    IncidentSeverity = "minor"
	SeverityMajor    IncidentSeverity = "major"
	SeverityCritical IncidentSeverity = "critical"
)

// IsValid checks if the severity is valid
func (s IncidentSeverity) IsValid() bool {
	switch s {
	case SeverityMinor, SeverityMajor, SeverityCritical:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (s IncidentSeverity) String() string {
	return string(s)
}

// NewIncidentSeverity creates and validates an IncidentSeverity
func NewIncidentSeverity(severity string) (IncidentSeverity, error) {
	s := IncidentSeverity(severity)
	if !s.IsValid() {
		return "", fmt.Errorf("invalid incident severity: %s", severity)
	}
	return s, nil
}

// IncidentStatus represents the current status of an incident
type IncidentStatus string

const (
	StatusInvestigating IncidentStatus = "investigating"
	StatusIdentified    IncidentStatus = "identified"
	StatusMonitoring    IncidentStatus = "monitoring"
	StatusResolved      IncidentStatus = "resolved"
)

// IsValid checks if the incident status is valid
func (s IncidentStatus) IsValid() bool {
	switch s {
	case StatusInvestigating, StatusIdentified, StatusMonitoring, StatusResolved:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (s IncidentStatus) String() string {
	return string(s)
}

// NewIncidentStatus creates and validates an IncidentStatus
func NewIncidentStatus(status string) (IncidentStatus, error) {
	s := IncidentStatus(status)
	if !s.IsValid() {
		return "", fmt.Errorf("invalid incident status: %s", status)
	}
	return s, nil
}

// IsResolved returns true if the incident is resolved
func (s IncidentStatus) IsResolved() bool {
	return s == StatusResolved
}

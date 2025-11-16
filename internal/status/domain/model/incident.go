package model

import (
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// Incident represents a service outage or degradation event
type Incident struct {
	IncidentID        uint64
	ServiceName       value.ServiceName
	Severity          value.IncidentSeverity
	Title             string
	Description       *string // nullable
	Status            value.IncidentStatus
	StartedAt         time.Time
	ResolvedAt        *time.Time // nullable
	DurationMinutes   *uint32    // nullable, calculated on resolution
	AffectedServices  []value.ServiceName
	CreatedAt         time.Time
	UpdatedAt         *time.Time // nullable
}

// NewIncident creates a new Incident
func NewIncident(
	serviceName value.ServiceName,
	severity value.IncidentSeverity,
	title string,
	description *string,
) *Incident {
	now := time.Now().UTC()
	return &Incident{
		ServiceName:      serviceName,
		Severity:         severity,
		Title:            title,
		Description:      description,
		Status:           value.StatusInvestigating,
		StartedAt:        now,
		AffectedServices: []value.ServiceName{serviceName},
		CreatedAt:        now,
	}
}

// Resolve marks the incident as resolved
func (i *Incident) Resolve() {
	now := time.Now().UTC()
	i.Status = value.StatusResolved
	i.ResolvedAt = &now
	i.UpdatedAt = &now

	// Calculate duration in minutes
	duration := uint32(now.Sub(i.StartedAt).Minutes())
	i.DurationMinutes = &duration
}

// UpdateStatus updates the incident status
func (i *Incident) UpdateStatus(status value.IncidentStatus) {
	i.Status = status
	now := time.Now().UTC()
	i.UpdatedAt = &now
}

// IsResolved returns true if the incident is resolved
func (i *Incident) IsResolved() bool {
	return i.Status.IsResolved()
}

// IsCritical returns true if the incident is critical
func (i *Incident) IsCritical() bool {
	return i.Severity == value.SeverityCritical
}

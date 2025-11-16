package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// IncidentRepository defines the interface for incident persistence
type IncidentRepository interface {
	// CreateIncident stores a new incident
	CreateIncident(ctx context.Context, incident *model.Incident) error

	// GetIncidentByID retrieves an incident by ID
	GetIncidentByID(ctx context.Context, incidentID uint64) (*model.Incident, error)

	// GetActiveIncidents retrieves all unresolved incidents
	GetActiveIncidents(ctx context.Context) ([]*model.Incident, error)

	// GetIncidentsByService retrieves incidents for a specific service
	GetIncidentsByService(ctx context.Context, serviceName value.ServiceName, limit int) ([]*model.Incident, error)

	// GetRecentIncidents retrieves the most recent incidents
	GetRecentIncidents(ctx context.Context, limit int) ([]*model.Incident, error)

	// UpdateIncident updates an existing incident
	UpdateIncident(ctx context.Context, incident *model.Incident) error

	// ResolveIncident marks an incident as resolved
	ResolveIncident(ctx context.Context, incidentID uint64) error
}

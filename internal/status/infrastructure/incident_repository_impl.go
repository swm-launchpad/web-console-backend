package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/status/infrastructure/sqlc"
)

// IncidentRepositoryImpl implements the IncidentRepository interface using SQLC
type IncidentRepositoryImpl struct {
	db      *sql.DB
	queries *sqlc.Queries
}

// NewIncidentRepository creates a new IncidentRepositoryImpl
func NewIncidentRepository(db *sql.DB) repository.IncidentRepository {
	return &IncidentRepositoryImpl{
		db:      db,
		queries: sqlc.New(db),
	}
}

// CreateIncident stores a new incident
func (r *IncidentRepositoryImpl) CreateIncident(ctx context.Context, incident *model.Incident) error {
	affectedServices, err := json.Marshal(incident.AffectedServices)
	if err != nil {
		return err
	}

	var description sql.NullString
	if incident.Description != nil {
		description = sql.NullString{String: *incident.Description, Valid: true}
	}

	_, err = r.queries.CreateIncident(ctx, sqlc.CreateIncidentParams{
		ServiceName:      incident.ServiceName.String(),
		Severity:         sqlc.ServiceIncidentsSeverity(incident.Severity.String()),
		Title:            incident.Title,
		Description:      description,
		Status:           sqlc.ServiceIncidentsStatus(incident.Status.String()),
		StartedAt:        incident.StartedAt,
		AffectedServices: affectedServices,
	})

	return err
}

// GetIncidentByID retrieves an incident by ID
func (r *IncidentRepositoryImpl) GetIncidentByID(ctx context.Context, incidentID uint64) (*model.Incident, error) {
	row, err := r.queries.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return nil, err
	}

	return r.rowToIncident(row)
}

// GetActiveIncidents retrieves all unresolved incidents
func (r *IncidentRepositoryImpl) GetActiveIncidents(ctx context.Context) ([]*model.Incident, error) {
	rows, err := r.queries.GetActiveIncidents(ctx)
	if err != nil {
		return nil, err
	}

	incidents := make([]*model.Incident, 0, len(rows))
	for _, row := range rows {
		incident, err := r.rowToIncident(row)
		if err != nil {
			continue
		}
		incidents = append(incidents, incident)
	}

	return incidents, nil
}

// GetIncidentsByService retrieves incidents for a specific service
func (r *IncidentRepositoryImpl) GetIncidentsByService(ctx context.Context, serviceName value.ServiceName, limit int) ([]*model.Incident, error) {
	rows, err := r.queries.GetIncidentsByService(ctx, sqlc.GetIncidentsByServiceParams{
		ServiceName: serviceName.String(),
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, err
	}

	incidents := make([]*model.Incident, 0, len(rows))
	for _, row := range rows {
		incident, err := r.rowToIncident(row)
		if err != nil {
			continue
		}
		incidents = append(incidents, incident)
	}

	return incidents, nil
}

// GetRecentIncidents retrieves the most recent incidents
func (r *IncidentRepositoryImpl) GetRecentIncidents(ctx context.Context, limit int) ([]*model.Incident, error) {
	rows, err := r.queries.GetRecentIncidents(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	incidents := make([]*model.Incident, 0, len(rows))
	for _, row := range rows {
		incident, err := r.rowToIncident(row)
		if err != nil {
			continue
		}
		incidents = append(incidents, incident)
	}

	return incidents, nil
}

// UpdateIncident updates an existing incident
func (r *IncidentRepositoryImpl) UpdateIncident(ctx context.Context, incident *model.Incident) error {
	affectedServices, err := json.Marshal(incident.AffectedServices)
	if err != nil {
		return err
	}

	var description sql.NullString
	if incident.Description != nil {
		description = sql.NullString{String: *incident.Description, Valid: true}
	}

	var resolvedAt sql.NullTime
	if incident.ResolvedAt != nil {
		resolvedAt = sql.NullTime{Time: *incident.ResolvedAt, Valid: true}
	}

	var durationMinutes sql.NullInt32
	if incident.DurationMinutes != nil {
		durationMinutes = sql.NullInt32{Int32: int32(*incident.DurationMinutes), Valid: true}
	}

	return r.queries.UpdateIncident(ctx, sqlc.UpdateIncidentParams{
		Severity:         sqlc.ServiceIncidentsSeverity(incident.Severity.String()),
		Title:            incident.Title,
		Description:      description,
		Status:           sqlc.ServiceIncidentsStatus(incident.Status.String()),
		ResolvedAt:       resolvedAt,
		DurationMinutes:  durationMinutes,
		AffectedServices: affectedServices,
		IncidentID:       incident.IncidentID,
	})
}

// ResolveIncident marks an incident as resolved
func (r *IncidentRepositoryImpl) ResolveIncident(ctx context.Context, incidentID uint64) error {
	now := time.Now().UTC()

	// Get the incident to calculate duration
	incident, err := r.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return err
	}

	duration := uint32(now.Sub(incident.StartedAt).Minutes())

	return r.queries.ResolveIncident(ctx, sqlc.ResolveIncidentParams{
		ResolvedAt:      sql.NullTime{Time: now, Valid: true},
		DurationMinutes: sql.NullInt32{Int32: int32(duration), Valid: true},
		IncidentID:      incidentID,
	})
}

// rowToIncident converts a SQLC row to a domain Incident
func (r *IncidentRepositoryImpl) rowToIncident(row sqlc.ServiceIncident) (*model.Incident, error) {
	serviceName, err := value.NewServiceName(row.ServiceName)
	if err != nil {
		return nil, err
	}

	severity, err := value.NewIncidentSeverity(string(row.Severity))
	if err != nil {
		return nil, err
	}

	status, err := value.NewIncidentStatus(string(row.Status))
	if err != nil {
		return nil, err
	}

	var description *string
	if row.Description.Valid {
		description = &row.Description.String
	}

	var resolvedAt *time.Time
	if row.ResolvedAt.Valid {
		resolvedAt = &row.ResolvedAt.Time
	}

	var durationMinutes *uint32
	if row.DurationMinutes.Valid {
		val := uint32(row.DurationMinutes.Int32)
		durationMinutes = &val
	}

	var affectedServices []value.ServiceName
	if len(row.AffectedServices) > 0 {
		var serviceNames []string
		if err := json.Unmarshal(row.AffectedServices, &serviceNames); err == nil {
			for _, name := range serviceNames {
				if svc, err := value.NewServiceName(name); err == nil {
					affectedServices = append(affectedServices, svc)
				}
			}
		}
	}
	if affectedServices == nil {
		affectedServices = []value.ServiceName{serviceName}
	}

	var updatedAt *time.Time
	if row.UpdatedAt.Valid {
		updatedAt = &row.UpdatedAt.Time
	}

	return &model.Incident{
		IncidentID:       uint64(row.IncidentID),
		ServiceName:      serviceName,
		Severity:         severity,
		Title:            row.Title,
		Description:      description,
		Status:           status,
		StartedAt:        row.StartedAt,
		ResolvedAt:       resolvedAt,
		DurationMinutes:  durationMinutes,
		AffectedServices: affectedServices,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        updatedAt,
	}, nil
}

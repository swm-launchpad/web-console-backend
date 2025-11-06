package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"go.uber.org/zap"
)

// CheckProjectPodStatusInput represents the input for checking project pod status
type CheckProjectPodStatusInput struct {
	ProjectID uint
}

// CheckProjectPodStatusOutput represents the output for checking project pod status
type CheckProjectPodStatusOutput struct {
	Exists          bool   `json:"exists"`
	Status          string `json:"status,omitempty"`
	Phase           string `json:"phase,omitempty"`
	Reason          string `json:"reason,omitempty"`
	ReadyContainers int    `json:"ready_containers,omitempty"`
	TotalContainers int    `json:"total_containers,omitempty"`
}

// CheckProjectPodStatusUseCase handles checking the pod status for a project
type CheckProjectPodStatusUseCase struct {
	kubeClient infrastructure.KubeClient
	logger     logger.Logger
}

// NewCheckProjectPodStatusUseCase creates a new CheckProjectPodStatusUseCase
func NewCheckProjectPodStatusUseCase(
	kubeClient infrastructure.KubeClient,
	logger logger.Logger,
) *CheckProjectPodStatusUseCase {
	return &CheckProjectPodStatusUseCase{
		kubeClient: kubeClient,
		logger:     logger,
	}
}

// Execute checks the pod status for a project
func (uc *CheckProjectPodStatusUseCase) Execute(ctx context.Context, input CheckProjectPodStatusInput) (*CheckProjectPodStatusOutput, error) {
	uc.logger.Info(ctx, "Checking project pod status",
		zap.Uint("project_id", input.ProjectID),
	)

	// Get pod status from Kubernetes
	podStatus, err := uc.kubeClient.GetProjectPodStatus(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get project pod status",
			zap.Uint("project_id", input.ProjectID),
			zap.Error(err),
		)
		return nil, err
	}

	output := &CheckProjectPodStatusOutput{
		Exists:          podStatus.Exists,
		Status:          podStatus.Status,
		Phase:           podStatus.Phase,
		Reason:          podStatus.Reason,
		ReadyContainers: podStatus.ReadyContainers,
		TotalContainers: podStatus.TotalContainers,
	}

	uc.logger.Info(ctx, "Project pod status retrieved",
		zap.Uint("project_id", input.ProjectID),
		zap.Bool("exists", output.Exists),
		zap.String("status", output.Status),
		zap.String("phase", output.Phase),
	)

	return output, nil
}

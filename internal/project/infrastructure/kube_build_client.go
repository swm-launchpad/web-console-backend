package infrastructure

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"go.uber.org/zap"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// kubeBuildClient implements the KubeBuildClient interface using Kubernetes client-go library.
// It uses Dynamic Client to interact with Tekton CRDs (PipelineRuns) in the build-pipeline namespace.
type kubeBuildClient struct {
	dynamicClient  dynamic.Interface
	namespace      string
	pipelineRunGVR schema.GroupVersionResource
	logger         logger.Logger
}

// NewKubeBuildClient creates a new Kubernetes build client using configuration from environment variables.
//
// Required environment variables:
//   - KUBE_API_SERVER: The Kubernetes API server URL (e.g., "https://kube-api.launchpad.kr:6443")
//   - KUBE_SERVICE_ACCOUNT_TOKEN: The ServiceAccount token for authentication
//   - KUBE_BUILD_NAMESPACE: The namespace where build PipelineRuns are deployed (e.g., "build-pipeline")
//   - KUBE_CA_CERT_PATH: Path to the CA certificate file for TLS verification (required)
//
// Returns an error if any required environment variable is missing or if the client
// cannot be initialized.
func NewKubeBuildClient(log logger.Logger) (infrastructure.KubeBuildClient, error) {
	// Read configuration from environment variables
	apiServer := os.Getenv("KUBE_API_SERVER")
	if apiServer == "" {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	token := os.Getenv("KUBE_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	namespace := os.Getenv("KUBE_BUILD_NAMESPACE")
	if namespace == "" {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	// Configure TLS - CA certificate is required
	caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
	if caCertPath == "" {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	// Verify CA cert file exists
	if _, err := os.Stat(caCertPath); err != nil {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	tlsConfig := rest.TLSClientConfig{
		CAFile: caCertPath,
	}

	// Create REST config with Bearer token authentication
	config := &rest.Config{
		Host:            apiServer,
		BearerToken:     token,
		TLSClientConfig: tlsConfig,
	}

	// Create dynamic client for CRDs (Tekton PipelineRuns)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	// Define Group-Version-Resource for Tekton PipelineRuns
	pipelineRunGVR := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1",
		Resource: "pipelineruns",
	}

	return &kubeBuildClient{
		dynamicClient:  dynamicClient,
		namespace:      namespace,
		pipelineRunGVR: pipelineRunGVR,
		logger:         log,
	}, nil
}

// GetPipelineRunStatus retrieves the current status of a build PipelineRun.
// It examines the PipelineRun's status.conditions and status.results to extract
// both status information and build results (latest_commit_hash, image_tag, should_build).
func (k *kubeBuildClient) GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
	k.logger.Info(ctx, "kube build client get pipeline run status started",
		zap.String("pipeline_run_name", pipelineRunName),
		zap.String("namespace", k.namespace),
	)

	// Get the PipelineRun resource
	pipelineRun, err := k.dynamicClient.
		Resource(k.pipelineRunGVR).
		Namespace(k.namespace).
		Get(ctx, pipelineRunName, metav1.GetOptions{})

	if err != nil {
		// Map Kubernetes NotFound error to domain error
		if apierrors.IsNotFound(err) {
			k.logger.Error(ctx, "kube build client pipeline run not found",
				zap.String("pipeline_run_name", pipelineRunName),
				zap.Error(projecterrors.ErrKubePipelineRunNotFound),
			)
			return nil, projecterrors.ErrKubePipelineRunNotFound
		}
		k.logger.Error(ctx, "kube build client get pipeline run failed",
			zap.String("pipeline_run_name", pipelineRunName),
			zap.Error(err),
		)
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	// Extract status information including results
	status := extractPipelineRunStatusWithResults(pipelineRun)

	k.logger.Info(ctx, "kube build client get pipeline run status completed",
		zap.String("pipeline_run_name", pipelineRunName),
		zap.String("status", status.Status),
		zap.String("reason", status.Reason),
	)

	return status, nil
}

// FindPipelineRunNameByEventID retrieves the PipelineRun name associated with a Tekton event ID.
// It searches for PipelineRuns with the "triggers.tekton.dev/triggers-eventid" label matching the given EventID.
func (k *kubeBuildClient) FindPipelineRunNameByEventID(ctx context.Context, eventID string) (string, error) {
	k.logger.Info(ctx, "kube build client find pipeline run by event id started",
		zap.String("event_id", eventID),
		zap.String("namespace", k.namespace),
	)

	// List PipelineRuns with triggers.tekton.dev/triggers-eventid label
	pipelineRuns, err := k.dynamicClient.
		Resource(k.pipelineRunGVR).
		Namespace(k.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("triggers.tekton.dev/triggers-eventid=%s", eventID),
		})

	if err != nil {
		k.logger.Error(ctx, "kube build client find pipeline run by event id failed",
			zap.String("event_id", eventID),
			zap.Error(err),
		)
		return "", projecterrors.ErrKubernetesUnavailable
	}

	// Check if any PipelineRuns were found
	if len(pipelineRuns.Items) == 0 {
		k.logger.Error(ctx, "kube build client no pipeline run found for event id",
			zap.String("event_id", eventID),
			zap.Error(projecterrors.ErrKubePipelineRunNotFound),
		)
		return "", projecterrors.ErrKubePipelineRunNotFound
	}

	k.logger.Info(ctx, "kube build client found pipeline runs for event id",
		zap.String("event_id", eventID),
		zap.Int("count", len(pipelineRuns.Items)),
	)

	// If multiple PipelineRuns found, return the most recent one
	if len(pipelineRuns.Items) > 1 {
		k.logger.Info(ctx, "kube build client multiple pipeline runs found, selecting most recent",
			zap.String("event_id", eventID),
			zap.Int("count", len(pipelineRuns.Items)),
		)

		// Extract status information to get startTime
		runs := make([]*dto.PipelineRun, 0, len(pipelineRuns.Items))
		for _, pr := range pipelineRuns.Items {
			run := extractPipelineRunStatusWithResults(&pr)
			runs = append(runs, run)
		}

		// Sort by creation time (newest first)
		sort.Slice(runs, func(i, j int) bool {
			// Both nil: equal, return false for strict ordering
			if runs[i].StartTime == nil && runs[j].StartTime == nil {
				return false
			}
			// i is nil: place after j (nil timestamps last)
			if runs[i].StartTime == nil {
				return false
			}
			// j is nil: place i before j
			if runs[j].StartTime == nil {
				return true
			}
			// Both have timestamps: newer first
			return runs[i].StartTime.After(*runs[j].StartTime)
		})

		// Return the name of the most recent PipelineRun
		k.logger.Info(ctx, "kube build client find pipeline run by event id completed",
			zap.String("event_id", eventID),
			zap.String("pipeline_run_name", runs[0].Name),
		)
		return runs[0].Name, nil
	}

	// Single PipelineRun found
	pipelineRunName := pipelineRuns.Items[0].GetName()
	k.logger.Info(ctx, "kube build client find pipeline run by event id completed",
		zap.String("event_id", eventID),
		zap.String("pipeline_run_name", pipelineRunName),
	)
	return pipelineRunName, nil
}

// extractPipelineRunStatusWithResults extracts status information and results from a
// PipelineRun unstructured object.
// It examines the status.conditions field to extract raw condition values,
// and also parses status.results to extract build output values.
// Prefers type="Succeeded" condition, but falls back to the first condition if not found.
func extractPipelineRunStatusWithResults(pr *unstructured.Unstructured) *dto.PipelineRun {
	pipelineRun := &dto.PipelineRun{
		Name:   pr.GetName(),
		Status: "Unknown",
		Reason: "Unknown",
	}

	// Extract status field
	statusField, found, err := unstructured.NestedMap(pr.Object, "status")
	if !found || err != nil {
		return pipelineRun
	}

	// Extract startTime
	// Tekton uses RFC3339Nano format (with nanosecond precision)
	startTimeStr, found, err := unstructured.NestedString(statusField, "startTime")
	if found && err == nil {
		if t := parseTimestamp(startTimeStr); t != nil {
			pipelineRun.StartTime = t
		}
	}

	// Extract completionTime
	completionTimeStr, found, err := unstructured.NestedString(statusField, "completionTime")
	if found && err == nil {
		if t := parseTimestamp(completionTimeStr); t != nil {
			pipelineRun.CompletionTime = t
		}
	}

	// Extract conditions
	conditions, found, err := unstructured.NestedSlice(statusField, "conditions")
	if found && err == nil {
		// Option A: Find type="Succeeded" condition first, fallback to first condition
		var selectedCondition map[string]interface{}

		// Try to find type="Succeeded" condition
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _, _ := unstructured.NestedString(condMap, "type")
			if condType == "Succeeded" {
				selectedCondition = condMap
				break
			}
		}

		// Fallback to first condition if "Succeeded" not found
		if selectedCondition == nil && len(conditions) > 0 {
			if condMap, ok := conditions[0].(map[string]interface{}); ok {
				selectedCondition = condMap
			}
		}

		// Extract raw values from selected condition (no transformation)
		if selectedCondition != nil {
			condStatus, _, _ := unstructured.NestedString(selectedCondition, "status")
			reason, _, _ := unstructured.NestedString(selectedCondition, "reason")
			message, _, _ := unstructured.NestedString(selectedCondition, "message")

			// Use raw values without transformation
			pipelineRun.Status = condStatus
			pipelineRun.Reason = reason
			pipelineRun.Message = message
		}
	}

	// Extract results from status.results
	// Build pipeline returns: latest_commit_hash, image_tag, should_build
	resultsSlice, found, err := unstructured.NestedSlice(statusField, "results")
	if found && err == nil {
		results := make(map[string]string)

		for _, resultItem := range resultsSlice {
			resultMap, ok := resultItem.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract name and value
			name, _, _ := unstructured.NestedString(resultMap, "name")
			value, _, _ := unstructured.NestedString(resultMap, "value")

			if name != "" {
				results[name] = value
			}
		}

		// Set results if any were found
		if len(results) > 0 {
			pipelineRun.Results = results
		}
	}

	return pipelineRun
}

// Compile-time assertion that kubeBuildClient implements KubeBuildClient interface
var _ infrastructure.KubeBuildClient = (*kubeBuildClient)(nil)

package infrastructure

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// kubeClient implements the KubeClient interface using Kubernetes client-go library.
// It uses Dynamic Client to interact with Tekton CRDs (PipelineRuns) and the standard
// Kubernetes client to retrieve Pod logs.
type kubeClient struct {
	dynamicClient  dynamic.Interface
	clientset      *kubernetes.Clientset
	namespace      string
	pipelineRunGVR schema.GroupVersionResource
	taskRunGVR     schema.GroupVersionResource
	logger         logger.Logger
}

// NewKubeDeployClient creates a new Kubernetes client using configuration from environment variables.
//
// Required environment variables:
//   - KUBE_API_SERVER: The Kubernetes API server URL (e.g., "https://kube-api.launchpad.kr:6443")
//   - KUBE_SERVICE_ACCOUNT_TOKEN: The ServiceAccount token for authentication
//   - KUBE_DEPLOY_NAMESPACE: The namespace where deploy PipelineRuns are deployed (e.g., "deploy-pipeline")
//   - KUBE_CA_CERT_PATH: Path to the CA certificate file for TLS verification (required)
//
// Returns an error if any required environment variable is missing or if the client
// cannot be initialized.
func NewKubeDeployClient(log logger.Logger) (infrastructure.KubeClient, error) {
	// Read configuration from environment variables
	apiServer := os.Getenv("KUBE_API_SERVER")
	if apiServer == "" {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	token := os.Getenv("KUBE_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	namespace := os.Getenv("KUBE_DEPLOY_NAMESPACE")
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

	// Create standard clientset for Pod operations
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	// Define Group-Version-Resource for Tekton PipelineRuns and TaskRuns
	pipelineRunGVR := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1",
		Resource: "pipelineruns",
	}

	taskRunGVR := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1",
		Resource: "taskruns",
	}

	return &kubeClient{
		dynamicClient:  dynamicClient,
		clientset:      clientset,
		namespace:      namespace,
		pipelineRunGVR: pipelineRunGVR,
		taskRunGVR:     taskRunGVR,
		logger:         log,
	}, nil
}

// GetPipelineRunStatus retrieves the current status of a PipelineRun.
// It examines the PipelineRun's status.conditions to extract raw condition values.
func (k *kubeClient) GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error) {
	k.logger.Info(ctx, "kube client get pipeline run status started",
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
			k.logger.Error(ctx, "kube client pipeline run not found",
				zap.String("pipeline_run_name", pipelineRunName),
				zap.Error(projecterrors.ErrKubePipelineRunNotFound),
			)
			return nil, projecterrors.ErrKubePipelineRunNotFound
		}
		k.logger.Error(ctx, "kube client get pipeline run failed",
			zap.String("pipeline_run_name", pipelineRunName),
			zap.Error(err),
		)
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	// Extract status information
	status := extractPipelineRunStatus(pipelineRun)

	k.logger.Info(ctx, "kube client get pipeline run status completed",
		zap.String("pipeline_run_name", pipelineRunName),
		zap.String("status", status.Status),
		zap.String("reason", status.Reason),
	)

	return status, nil
}

// GetPipelineRunLogs retrieves aggregated logs from all tasks in a PipelineRun.
// It traverses TaskRuns associated with the PipelineRun and collects logs from their Pods.
func (k *kubeClient) GetPipelineRunLogs(ctx context.Context, pipelineRunName string) (string, error) {
	k.logger.Info(ctx, "kube client get pipeline run logs started",
		zap.String("pipeline_run_name", pipelineRunName),
		zap.String("namespace", k.namespace),
	)

	// First, verify that the PipelineRun exists
	// This ensures we return ErrKubePipelineRunNotFound for nonexistent PipelineRuns,
	// distinguishing "not found" from "no logs available yet"
	_, err := k.dynamicClient.
		Resource(k.pipelineRunGVR).
		Namespace(k.namespace).
		Get(ctx, pipelineRunName, metav1.GetOptions{})

	if err != nil {
		// Map Kubernetes NotFound error to domain error
		if apierrors.IsNotFound(err) {
			k.logger.Error(ctx, "kube client pipeline run not found for logs",
				zap.String("pipeline_run_name", pipelineRunName),
				zap.Error(projecterrors.ErrKubePipelineRunNotFound),
			)
			return "", projecterrors.ErrKubePipelineRunNotFound
		}
		k.logger.Error(ctx, "kube client verify pipeline run failed",
			zap.String("pipeline_run_name", pipelineRunName),
			zap.Error(err),
		)
		return "", projecterrors.ErrKubernetesUnavailable
	}

	// List TaskRuns that belong to this PipelineRun
	taskRuns, err := k.dynamicClient.
		Resource(k.taskRunGVR).
		Namespace(k.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", pipelineRunName),
		})

	if err != nil {
		k.logger.Error(ctx, "kube client list task runs failed",
			zap.String("pipeline_run_name", pipelineRunName),
			zap.Error(err),
		)
		return "", projecterrors.ErrKubernetesUnavailable
	}

	k.logger.Info(ctx, "kube client found task runs",
		zap.String("pipeline_run_name", pipelineRunName),
		zap.Int("task_run_count", len(taskRuns.Items)),
	)

	// Collect logs from all TaskRuns
	var logs strings.Builder
	for _, taskRun := range taskRuns.Items {
		taskRunName := taskRun.GetName()
		taskName := getTaskNameFromTaskRun(&taskRun)

		// Get the Pod name from TaskRun status
		// Tekton stores the actual Pod name in .status.podName
		podName, err := getPodNameFromTaskRun(&taskRun)
		if err != nil {
			k.logger.Error(ctx, "kube client failed to get pod name from task run",
				zap.String("task_run_name", taskRunName),
				zap.Error(err),
			)
			logs.WriteString(fmt.Sprintf("\n=== Task: %s (TaskRun: %s) ===\n", taskName, taskRunName))
			logs.WriteString(fmt.Sprintf("Error getting pod name: %v\n", err))
			continue
		}

		taskLogs, err := k.getPodLogs(ctx, podName)
		if err != nil {
			// Log the error but continue with other tasks
			k.logger.Error(ctx, "kube client failed to get pod logs",
				zap.String("pod_name", podName),
				zap.String("task_run_name", taskRunName),
				zap.Error(err),
			)
			logs.WriteString(fmt.Sprintf("\n=== Task: %s (TaskRun: %s) ===\n", taskName, taskRunName))
			logs.WriteString(fmt.Sprintf("Error retrieving logs: %v\n", err))
			continue
		}

		logs.WriteString(fmt.Sprintf("\n=== Task: %s (TaskRun: %s) ===\n", taskName, taskRunName))
		logs.WriteString(taskLogs)
	}

	if logs.Len() == 0 {
		k.logger.Info(ctx, "kube client get pipeline run logs completed (no logs)",
			zap.String("pipeline_run_name", pipelineRunName),
		)
		return "No logs available", nil
	}

	k.logger.Info(ctx, "kube client get pipeline run logs completed",
		zap.String("pipeline_run_name", pipelineRunName),
		zap.Int("log_size", logs.Len()),
	)

	return logs.String(), nil
}

// ListPipelineRuns retrieves all PipelineRuns associated with a project.
// It filters PipelineRuns by the "project-id" label and sorts them by creation time
// in descending order (newest first).
func (k *kubeClient) ListPipelineRuns(ctx context.Context, projectID uint) ([]*dto.PipelineRun, error) {
	k.logger.Info(ctx, "kube client list pipeline runs started",
		zap.Uint("project_id", projectID),
		zap.String("namespace", k.namespace),
	)

	// List PipelineRuns with project-id label
	pipelineRuns, err := k.dynamicClient.
		Resource(k.pipelineRunGVR).
		Namespace(k.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("project-id=%d", projectID),
		})

	if err != nil {
		k.logger.Error(ctx, "kube client list pipeline runs failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return nil, projecterrors.ErrKubernetesUnavailable
	}

	// Convert to PipelineRun
	result := make([]*dto.PipelineRun, 0, len(pipelineRuns.Items))
	for _, pr := range pipelineRuns.Items {
		// Extract status information using the shared function
		pipelineRun := extractPipelineRunStatus(&pr)

		// Set ProjectID (common for all results)
		pipelineRun.ProjectID = projectID

		// Extract EventID from labels
		labels := pr.GetLabels()
		if eventID, ok := labels["triggers.tekton.dev/triggers-eventid"]; ok {
			pipelineRun.EventID = eventID
		}

		result = append(result, pipelineRun)
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		// Both nil: equal, return false for strict ordering
		if result[i].StartTime == nil && result[j].StartTime == nil {
			return false
		}
		// i is nil: place after j (nil timestamps last)
		if result[i].StartTime == nil {
			return false
		}
		// j is nil: place i before j
		if result[j].StartTime == nil {
			return true
		}
		// Both have timestamps: newer first
		return result[i].StartTime.After(*result[j].StartTime)
	})

	k.logger.Info(ctx, "kube client list pipeline runs completed",
		zap.Uint("project_id", projectID),
		zap.Int("count", len(result)),
	)

	return result, nil
}

// FindPipelineRunNameByEventID retrieves the PipelineRun name associated with a Tekton event ID.
// It searches for PipelineRuns with the "triggers.tekton.dev/triggers-eventid" label matching the given EventID.
func (k *kubeClient) FindPipelineRunNameByEventID(ctx context.Context, eventID string) (string, error) {
	k.logger.Info(ctx, "kube client find pipeline run by event id started",
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
		k.logger.Error(ctx, "kube client find pipeline run by event id failed",
			zap.String("event_id", eventID),
			zap.Error(err),
		)
		return "", projecterrors.ErrKubernetesUnavailable
	}

	// Check if any PipelineRuns were found
	if len(pipelineRuns.Items) == 0 {
		k.logger.Error(ctx, "kube client no pipeline run found for event id",
			zap.String("event_id", eventID),
			zap.Error(projecterrors.ErrKubePipelineRunNotFound),
		)
		return "", projecterrors.ErrKubePipelineRunNotFound
	}

	k.logger.Info(ctx, "kube client found pipeline runs for event id",
		zap.String("event_id", eventID),
		zap.Int("count", len(pipelineRuns.Items)),
	)

	// If multiple PipelineRuns found, return the most recent one
	if len(pipelineRuns.Items) > 1 {
		k.logger.Info(ctx, "kube client multiple pipeline runs found, selecting most recent",
			zap.String("event_id", eventID),
			zap.Int("count", len(pipelineRuns.Items)),
		)

		// Extract status information to get startTime
		runs := make([]*dto.PipelineRun, 0, len(pipelineRuns.Items))
		for _, pr := range pipelineRuns.Items {
			run := extractPipelineRunStatus(&pr)
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
		k.logger.Info(ctx, "kube client find pipeline run by event id completed",
			zap.String("event_id", eventID),
			zap.String("pipeline_run_name", runs[0].Name),
		)
		return runs[0].Name, nil
	}

	// Single PipelineRun found
	pipelineRunName := pipelineRuns.Items[0].GetName()
	k.logger.Info(ctx, "kube client find pipeline run by event id completed",
		zap.String("event_id", eventID),
		zap.String("pipeline_run_name", pipelineRunName),
	)
	return pipelineRunName, nil
}

// extractPipelineRunStatus extracts status information from a PipelineRun unstructured object.
// It examines the status.conditions field to extract raw condition values.
// Prefers type="Succeeded" condition, but falls back to the first condition if not found.
func extractPipelineRunStatus(pr *unstructured.Unstructured) *dto.PipelineRun {
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
	if !found || err != nil {
		return pipelineRun
	}

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

	return pipelineRun
}

// getTaskNameFromTaskRun extracts the task name from a TaskRun resource.
func getTaskNameFromTaskRun(taskRun *unstructured.Unstructured) string {
	// Try to get from label
	labels := taskRun.GetLabels()
	if taskName, ok := labels["tekton.dev/pipelineTask"]; ok {
		return taskName
	}

	// Fallback to TaskRun name
	return taskRun.GetName()
}

// getPodNameFromTaskRun extracts the Pod name from a TaskRun resource.
// Tekton stores the actual Pod name in .status.podName field.
func getPodNameFromTaskRun(taskRun *unstructured.Unstructured) (string, error) {
	// Try to get from status.podName
	podName, found, err := unstructured.NestedString(taskRun.Object, "status", "podName")
	if err != nil {
		return "", projecterrors.ErrKubernetesUnavailable
	}
	if !found || podName == "" {
		return "", projecterrors.ErrKubernetesUnavailable
	}

	return podName, nil
}

// parseTimestamp parses a Kubernetes/Tekton timestamp string.
// Tekton uses RFC3339Nano format (with nanosecond precision).
// This function tries RFC3339Nano first, then falls back to RFC3339.
func parseTimestamp(timeStr string) *time.Time {
	// Try RFC3339Nano first (e.g., "2025-10-07T12:34:56.123456789Z")
	if t, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
		return &t
	}

	// Fallback to RFC3339 (e.g., "2025-10-07T12:34:56Z")
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return &t
	}

	// Unable to parse
	return nil
}

// getPodLogs retrieves logs from all containers in a Pod.
func (k *kubeClient) getPodLogs(ctx context.Context, podName string) (string, error) {
	// Get Pod to list containers
	pod, err := k.clientset.CoreV1().Pods(k.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", projecterrors.ErrKubernetesUnavailable
	}

	var logs strings.Builder

	// Get logs from each container
	for _, container := range pod.Spec.Containers {
		// Skip sidecar containers (Tekton internal)
		if strings.HasPrefix(container.Name, "sidecar-") {
			continue
		}

		containerLogs, err := k.getContainerLogs(ctx, podName, container.Name)
		if err != nil {
			logs.WriteString(fmt.Sprintf("\n--- Container: %s ---\n", container.Name))
			logs.WriteString(fmt.Sprintf("Error: %v\n", err))
			continue
		}

		logs.WriteString(fmt.Sprintf("\n--- Container: %s ---\n", container.Name))
		logs.WriteString(containerLogs)
	}

	return logs.String(), nil
}

// getContainerLogs retrieves logs from a specific container in a Pod.
func (k *kubeClient) getContainerLogs(ctx context.Context, podName, containerName string) (string, error) {
	req := k.clientset.CoreV1().Pods(k.namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return "", projecterrors.ErrKubernetesUnavailable
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			// Log close error but don't override the main error
			_ = closeErr
		}
	}()

	// Read all logs
	var logs strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			logs.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return logs.String(), nil
}

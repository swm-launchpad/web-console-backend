package infrastructure

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"

	corev1 "k8s.io/api/core/v1"
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
}

// NewKubeClient creates a new Kubernetes client using configuration from environment variables.
//
// Required environment variables:
//   - KUBE_API_SERVER: The Kubernetes API server URL (e.g., "https://kube-api.launchpad.kr")
//   - KUBE_SERVICE_ACCOUNT_TOKEN: The ServiceAccount token for authentication
//   - KUBE_DEPLOY_NAMESPACE: The namespace where PipelineRuns are deployed (e.g., "deploy-pipeline")
//
// Returns an error if any required environment variable is missing or if the client
// cannot be initialized.
func NewKubeClient() (infrastructure.KubeClient, error) {
	// Read configuration from environment variables
	apiServer := os.Getenv("KUBE_API_SERVER")
	if apiServer == "" {
		return nil, fmt.Errorf("KUBE_API_SERVER environment variable is required")
	}

	token := os.Getenv("KUBE_SERVICE_ACCOUNT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("KUBE_SERVICE_ACCOUNT_TOKEN environment variable is required")
	}

	namespace := os.Getenv("KUBE_DEPLOY_NAMESPACE")
	if namespace == "" {
		return nil, fmt.Errorf("KUBE_DEPLOY_NAMESPACE environment variable is required")
	}

	// Create REST config with Bearer token authentication
	config := &rest.Config{
		Host:        apiServer,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false, // We trust the server certificate
		},
	}

	// Create dynamic client for CRDs (Tekton PipelineRuns)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create standard clientset for Pod operations
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Define Group-Version-Resource for Tekton PipelineRuns and TaskRuns
	pipelineRunGVR := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1beta1",
		Resource: "pipelineruns",
	}

	taskRunGVR := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1beta1",
		Resource: "taskruns",
	}

	return &kubeClient{
		dynamicClient:  dynamicClient,
		clientset:      clientset,
		namespace:      namespace,
		pipelineRunGVR: pipelineRunGVR,
		taskRunGVR:     taskRunGVR,
	}, nil
}

// GetPipelineRunStatus retrieves the current status of a PipelineRun.
// It examines the PipelineRun's status.conditions to determine if it has
// succeeded, failed, or is still running.
func (k *kubeClient) GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRunStatus, error) {
	// Get the PipelineRun resource
	pipelineRun, err := k.dynamicClient.
		Resource(k.pipelineRunGVR).
		Namespace(k.namespace).
		Get(ctx, pipelineRunName, metav1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to get PipelineRun %s: %w", pipelineRunName, err)
	}

	// Extract status information
	status := extractPipelineRunStatus(pipelineRun)
	return status, nil
}

// GetPipelineRunLogs retrieves aggregated logs from all tasks in a PipelineRun.
// It traverses TaskRuns associated with the PipelineRun and collects logs from their Pods.
func (k *kubeClient) GetPipelineRunLogs(ctx context.Context, pipelineRunName string) (string, error) {
	// List TaskRuns that belong to this PipelineRun
	taskRuns, err := k.dynamicClient.
		Resource(k.taskRunGVR).
		Namespace(k.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", pipelineRunName),
		})

	if err != nil {
		return "", fmt.Errorf("failed to list TaskRuns for PipelineRun %s: %w", pipelineRunName, err)
	}

	// Collect logs from all TaskRuns
	var logs strings.Builder
	for _, taskRun := range taskRuns.Items {
		taskRunName := taskRun.GetName()
		taskName := getTaskNameFromTaskRun(&taskRun)

		// Get the Pod name from TaskRun status
		// Tekton stores the actual Pod name in .status.podName
		podName, err := getPodNameFromTaskRun(&taskRun)
		if err != nil {
			logs.WriteString(fmt.Sprintf("\n=== Task: %s (TaskRun: %s) ===\n", taskName, taskRunName))
			logs.WriteString(fmt.Sprintf("Error getting pod name: %v\n", err))
			continue
		}

		taskLogs, err := k.getPodLogs(ctx, podName)
		if err != nil {
			// Log the error but continue with other tasks
			logs.WriteString(fmt.Sprintf("\n=== Task: %s (TaskRun: %s) ===\n", taskName, taskRunName))
			logs.WriteString(fmt.Sprintf("Error retrieving logs: %v\n", err))
			continue
		}

		logs.WriteString(fmt.Sprintf("\n=== Task: %s (TaskRun: %s) ===\n", taskName, taskRunName))
		logs.WriteString(taskLogs)
	}

	if logs.Len() == 0 {
		return "No logs available", nil
	}

	return logs.String(), nil
}

// ListPipelineRuns retrieves all PipelineRuns associated with a project.
// It filters PipelineRuns by the "project-id" label and sorts them by creation time
// in descending order (newest first).
func (k *kubeClient) ListPipelineRuns(ctx context.Context, projectID uint) ([]*dto.PipelineRunInfo, error) {
	// List PipelineRuns with project-id label
	pipelineRuns, err := k.dynamicClient.
		Resource(k.pipelineRunGVR).
		Namespace(k.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("project-id=%d", projectID),
		})

	if err != nil {
		return nil, fmt.Errorf("failed to list PipelineRuns for project %d: %w", projectID, err)
	}

	// Convert to PipelineRunInfo
	result := make([]*dto.PipelineRunInfo, 0, len(pipelineRuns.Items))
	for _, pr := range pipelineRuns.Items {
		info := &dto.PipelineRunInfo{
			Name:      pr.GetName(),
			ProjectID: projectID,
		}

		// Extract status
		status := extractPipelineRunStatus(&pr)
		info.Status = status.Status
		info.StartTime = status.StartTime
		info.CompletionTime = status.CompletionTime
		info.Message = status.Message

		result = append(result, info)
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

	return result, nil
}

// extractPipelineRunStatus extracts status information from a PipelineRun unstructured object.
// It examines the status.conditions field to determine the current state.
func extractPipelineRunStatus(pr *unstructured.Unstructured) *dto.PipelineRunStatus {
	status := &dto.PipelineRunStatus{
		Name:   pr.GetName(),
		Status: "Unknown",
		Reason: "Unknown",
	}

	// Extract status field
	statusField, found, err := unstructured.NestedMap(pr.Object, "status")
	if !found || err != nil {
		return status
	}

	// Extract startTime
	// Tekton uses RFC3339Nano format (with nanosecond precision)
	startTimeStr, found, err := unstructured.NestedString(statusField, "startTime")
	if found && err == nil {
		if t := parseTimestamp(startTimeStr); t != nil {
			status.StartTime = t
		}
	}

	// Extract completionTime
	completionTimeStr, found, err := unstructured.NestedString(statusField, "completionTime")
	if found && err == nil {
		if t := parseTimestamp(completionTimeStr); t != nil {
			status.CompletionTime = t
		}
	}

	// Extract conditions
	conditions, found, err := unstructured.NestedSlice(statusField, "conditions")
	if !found || err != nil {
		return status
	}

	// Find the "Succeeded" condition
	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condMap, "type")
		if condType != "Succeeded" {
			continue
		}

		// Extract status (True/False/Unknown)
		condStatus, _, _ := unstructured.NestedString(condMap, "status")
		reason, _, _ := unstructured.NestedString(condMap, "reason")
		message, _, _ := unstructured.NestedString(condMap, "message")

		status.Reason = reason
		status.Message = message

		switch condStatus {
		case string(corev1.ConditionTrue):
			status.Status = "Succeeded"
		case string(corev1.ConditionFalse):
			// Check reason to distinguish between Failed and Cancelled
			if strings.Contains(strings.ToLower(reason), "cancel") ||
				strings.Contains(strings.ToLower(message), "cancel") {
				status.Status = "Cancelled"
			} else {
				status.Status = "Failed"
			}
		case string(corev1.ConditionUnknown):
			// PipelineRun is still running or pending
			if status.StartTime != nil {
				status.Status = "Running"
			} else {
				status.Status = "Pending"
			}
		}
	}

	return status
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
		return "", fmt.Errorf("failed to extract podName from TaskRun status: %w", err)
	}
	if !found || podName == "" {
		return "", fmt.Errorf("podName not found in TaskRun status")
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
		return "", fmt.Errorf("failed to get pod %s: %w", podName, err)
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
		return "", fmt.Errorf("failed to stream logs: %w", err)
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

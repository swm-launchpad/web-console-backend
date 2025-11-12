package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure"
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

// tektonNodePortClient implements the TektonNodePortClient interface
type tektonNodePortClient struct {
	tektonURL      string
	authHeader     string
	httpClient     *http.Client
	dynamicClient  dynamic.Interface
	clientset      kubernetes.Interface
	pipelineRunGVR schema.GroupVersionResource
	configMapNS    string // namespace where temporary-nodeport-config exists
	logger         logger.Logger
}

// NewTektonNodePortClient creates a new Tekton NodePort client
//
// Required environment variables:
//   - TEKTON_NODEPORT_API_URL: The Tekton EventListener endpoint URL
//   - TEKTON_API_AUTH: The Basic authentication header value
//   - KUBE_API_SERVER: The Kubernetes API server URL
//   - KUBE_SERVICE_ACCOUNT_TOKEN: The ServiceAccount token for authentication
//   - KUBE_CA_CERT_PATH: Path to the CA certificate file for TLS verification
//   - KUBE_NODEPORT_CONFIG_NAMESPACE: Namespace where temporary-nodeport-config ConfigMap exists
func NewTektonNodePortClient(log logger.Logger) (infrastructure.TektonNodePortClient, error) {
	ctx := context.Background()

	// Read Tekton configuration
	tektonURL := os.Getenv("TEKTON_NODEPORT_API_URL")
	log.Debug(ctx, "tekton nodeport client initialization - checking TEKTON_NODEPORT_API_URL",
		zap.String("value", tektonURL),
		zap.Bool("is_empty", tektonURL == ""),
	)
	if tektonURL == "" {
		log.Error(ctx, "TEKTON_NODEPORT_API_URL is not set")
		return nil, containererrors.ErrTektonUnavailable
	}

	authHeader := os.Getenv("TEKTON_API_AUTH")
	log.Debug(ctx, "tekton nodeport client initialization - checking TEKTON_API_AUTH",
		zap.Int("length", len(authHeader)),
		zap.Bool("is_empty", authHeader == ""),
	)
	if authHeader == "" {
		log.Error(ctx, "TEKTON_API_AUTH is not set")
		return nil, containererrors.ErrTektonUnavailable
	}

	// Read Kubernetes configuration
	apiServer := os.Getenv("KUBE_API_SERVER")
	log.Debug(ctx, "tekton nodeport client initialization - checking KUBE_API_SERVER",
		zap.String("value", apiServer),
		zap.Bool("is_empty", apiServer == ""),
	)
	if apiServer == "" {
		log.Error(ctx, "KUBE_API_SERVER is not set")
		return nil, containererrors.ErrKubernetesUnavailable
	}

	token := os.Getenv("KUBE_SERVICE_ACCOUNT_TOKEN")
	log.Debug(ctx, "tekton nodeport client initialization - checking KUBE_SERVICE_ACCOUNT_TOKEN",
		zap.Int("length", len(token)),
		zap.Bool("is_empty", token == ""),
	)
	if token == "" {
		log.Error(ctx, "KUBE_SERVICE_ACCOUNT_TOKEN is not set")
		return nil, containererrors.ErrKubernetesUnavailable
	}

	caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
	log.Debug(ctx, "tekton nodeport client initialization - checking KUBE_CA_CERT_PATH",
		zap.String("value", caCertPath),
		zap.Bool("is_empty", caCertPath == ""),
	)
	if caCertPath == "" {
		log.Error(ctx, "KUBE_CA_CERT_PATH is not set")
		return nil, containererrors.ErrKubernetesUnavailable
	}

	// Verify CA cert file exists
	log.Debug(ctx, "tekton nodeport client initialization - verifying CA cert file exists",
		zap.String("path", caCertPath),
	)
	if _, err := os.Stat(caCertPath); err != nil {
		log.Error(ctx, "CA cert file does not exist or is not accessible",
			zap.String("path", caCertPath),
			zap.Error(err),
		)
		return nil, containererrors.ErrKubernetesUnavailable
	}

	configMapNS := os.Getenv("KUBE_NODEPORT_CONFIG_NAMESPACE")
	if configMapNS == "" {
		configMapNS = "temporary-nodeport-pipeline" // default
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create REST config for Kubernetes
	tlsConfig := rest.TLSClientConfig{
		CAFile: caCertPath,
	}

	config := &rest.Config{
		Host:            apiServer,
		BearerToken:     token,
		TLSClientConfig: tlsConfig,
	}

	// Create dynamic client for PipelineRuns
	log.Debug(ctx, "tekton nodeport client initialization - creating dynamic client for PipelineRuns")
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Error(ctx, "failed to create dynamic client for PipelineRuns",
			zap.Error(err),
			zap.String("api_server", apiServer),
		)
		return nil, containererrors.ErrKubernetesUnavailable
	}

	// Create standard clientset for Services and ConfigMaps
	log.Debug(ctx, "tekton nodeport client initialization - creating standard clientset")
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(ctx, "failed to create standard clientset",
			zap.Error(err),
			zap.String("api_server", apiServer),
		)
		return nil, containererrors.ErrKubernetesUnavailable
	}

	// Define PipelineRun GVR
	pipelineRunGVR := schema.GroupVersionResource{
		Group:    "tekton.dev",
		Version:  "v1",
		Resource: "pipelineruns",
	}

	log.Info(ctx, "tekton nodeport client initialized successfully",
		zap.String("tekton_url", tektonURL),
		zap.String("api_server", apiServer),
		zap.String("config_map_namespace", configMapNS),
	)

	return &tektonNodePortClient{
		tektonURL:      tektonURL,
		authHeader:     authHeader,
		httpClient:     httpClient,
		dynamicClient:  dynamicClient,
		clientset:      clientset,
		pipelineRunGVR: pipelineRunGVR,
		configMapNS:    configMapNS,
		logger:         log,
	}, nil
}

// TriggerNodePortCreation triggers the Tekton pipeline to create a temporary NodePort
// Returns the PipelineRun name immediately without waiting for completion
func (c *tektonNodePortClient) TriggerNodePortCreation(ctx context.Context, params infrastructure.NodePortCreationParams) (string, error) {
	c.logger.Info(ctx, "tekton nodeport client trigger creation started",
		zap.String("service_name", params.ServiceName),
		zap.String("namespace", params.Namespace),
		zap.Int("target_port", params.TargetPort),
		zap.Int("ttl_seconds", params.TTLSeconds),
	)

	// Prepare request payload
	payload := map[string]interface{}{
		"service_name": params.ServiceName,
		"namespace":    params.Namespace,
		"target_port":  params.TargetPort,
		"ttl_seconds":  params.TTLSeconds,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error(ctx, "tekton nodeport client request marshaling failed", zap.Error(err))
		return "", containererrors.ErrInvalidRequest
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tektonURL, bytes.NewBuffer(requestBody))
	if err != nil {
		c.logger.Error(ctx, "tekton nodeport client http request creation failed", zap.Error(err))
		return "", containererrors.ErrTektonUnavailable
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader)

	// Send request
	c.logger.Info(ctx, "tekton nodeport client sending http request", zap.String("url", c.tektonURL))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(ctx, "tekton nodeport client http request failed", zap.Error(err))
		return "", containererrors.ErrTektonUnavailable
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn(ctx, "failed to close response body", zap.Error(closeErr))
		}
	}()

	// Read response body
	respBody, _ := io.ReadAll(resp.Body)

	// Log response headers for debugging
	c.logger.Info(ctx, "tekton nodeport client received response",
		zap.Int("status_code", resp.StatusCode),
		zap.Any("headers", resp.Header),
		zap.String("response_body", string(respBody)),
	)

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.Error(ctx, "tekton nodeport client http request returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(respBody)),
		)
		return "", containererrors.ErrTektonPipelineFailed
	}

	// Parse response body to extract event ID
	var responseData struct {
		EventID string `json:"eventID"`
	}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		c.logger.Error(ctx, "failed to parse tekton response body",
			zap.Error(err),
			zap.String("response_body", string(respBody)),
		)
		return "", containererrors.ErrTektonUnavailable
	}

	eventID := responseData.EventID
	if eventID == "" {
		c.logger.Error(ctx, "eventID not found in response body",
			zap.String("response_body", string(respBody)),
		)
		return "", containererrors.ErrTektonUnavailable
	}

	c.logger.Info(ctx, "tekton nodeport client pipeline triggered successfully",
		zap.Int("status_code", resp.StatusCode),
		zap.String("event_id", eventID),
	)

	return eventID, nil
}

// GetPipelineRunResult retrieves the result of a PipelineRun by its Tekton event ID
// Returns NodePortInfo if the PipelineRun has completed successfully
// Returns ErrNodePortNotFound if no PipelineRun exists
// Returns nil if PipelineRun is still running (no error, but no info either)
func (c *tektonNodePortClient) GetPipelineRunResult(ctx context.Context, eventID string) (*infrastructure.NodePortInfo, error) {
	c.logger.Info(ctx, "tekton nodeport client get pipeline result started",
		zap.String("event_id", eventID),
	)

	// Find PipelineRun by Tekton event ID label
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("triggers.tekton.dev/triggers-eventid=%s", eventID),
	}

	pipelineRuns, err := c.dynamicClient.
		Resource(c.pipelineRunGVR).
		Namespace(c.configMapNS).
		List(ctx, listOptions)

	if err != nil {
		c.logger.Error(ctx, "failed to list pipelineruns", zap.Error(err))
		return nil, containererrors.ErrTektonUnavailable
	}

	if len(pipelineRuns.Items) == 0 {
		c.logger.Info(ctx, "no pipelineruns found for event id", zap.String("event_id", eventID))
		return nil, containererrors.ErrNodePortNotFound
	}

	// Get the first (and should be only) PipelineRun
	latestRun := &pipelineRuns.Items[0]

	// Check status
	status, found, err := unstructured.NestedMap(latestRun.Object, "status")
	if err != nil || !found {
		c.logger.Debug(ctx, "pipelinerun status not available yet - still creating")
		return nil, nil // Still creating, no error
	}

	conditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil || !found || len(conditions) == 0 {
		c.logger.Debug(ctx, "pipelinerun conditions not available yet - still creating")
		return nil, nil // Still creating, no error
	}

	// Check latest condition
	latestCondition := conditions[0].(map[string]interface{})
	reason, _, _ := unstructured.NestedString(latestCondition, "reason")

	switch reason {
	case "Succeeded":
		// Extract result from pipelineResults
		c.logger.Info(ctx, "pipelinerun succeeded, extracting result")

		// Log the entire status structure for debugging
		statusJSON, _ := json.Marshal(status)
		c.logger.Info(ctx, "pipelinerun status structure",
			zap.String("status_json", string(statusJSON)),
		)

		// Extract metadata from PipelineRun labels
		labels, _, _ := unstructured.NestedStringMap(latestRun.Object, "metadata", "labels")
		serviceName := labels["knative-service"]
		namespace := labels["namespace"]

		params := infrastructure.NodePortCreationParams{
			ServiceName: serviceName,
			Namespace:   namespace,
		}
		return c.extractNodePortInfo(ctx, status, params)
	case "Failed":
		message, _, _ := unstructured.NestedString(latestCondition, "message")
		c.logger.Warn(ctx, "pipelinerun failed",
			zap.String("reason", reason),
			zap.String("message", message),
		)
		// Return failed status with NULL values
		return &infrastructure.NodePortInfo{
			ServiceName: "",
			Namespace:   "",
			Host:        "",
			Port:        0,
			TargetPort:  0,
			Protocol:    "tcp",
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now(),
			Status:      "failed",
		}, nil
	default:
		// Still running
		c.logger.Debug(ctx, "pipelinerun still running", zap.String("reason", reason))
		return nil, nil // Still running, no error
	}
}

// extractNodePortInfo extracts NodePort information from PipelineRun result
func (c *tektonNodePortClient) extractNodePortInfo(ctx context.Context, status map[string]interface{}, params infrastructure.NodePortCreationParams) (*infrastructure.NodePortInfo, error) {
	results, found, err := unstructured.NestedSlice(status, "results")
	if err != nil || !found {
		c.logger.Error(ctx, "results not found in status")
		return nil, containererrors.ErrTektonPipelineFailed
	}

	// Find nodeport_info result
	for _, result := range results {
		resultMap := result.(map[string]interface{})
		name, _, _ := unstructured.NestedString(resultMap, "name")
		if name == "nodeport_info" {
			valueStr, _, _ := unstructured.NestedString(resultMap, "value")

			// Parse JSON result
			var resultData map[string]interface{}
			if err := json.Unmarshal([]byte(valueStr), &resultData); err != nil {
				c.logger.Error(ctx, "failed to parse nodeport_info result", zap.Error(err))
				return nil, containererrors.ErrTektonPipelineFailed
			}

			// Extract fields
			host, _ := resultData["host"].(string)
			port, _ := resultData["port"].(float64)
			protocol, _ := resultData["protocol"].(string)
			serviceName, _ := resultData["service_name"].(string)
			createdAtStr, _ := resultData["created_at"].(string)
			expiresAtStr, _ := resultData["expires_at"].(string)

			createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
			expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)

			return &infrastructure.NodePortInfo{
				ServiceName: serviceName,
				Namespace:   params.Namespace,
				Host:        host,
				Port:        int(port),
				TargetPort:  params.TargetPort,
				Protocol:    protocol,
				CreatedAt:   createdAt,
				ExpiresAt:   expiresAt,
				Status:      "active",
			}, nil
		}
	}

	c.logger.Error(ctx, "nodeport_info result not found in results")
	return nil, containererrors.ErrTektonPipelineFailed
}

// GetNodePortService retrieves the current status of a NodePort service
func (c *tektonNodePortClient) GetNodePortService(ctx context.Context, serviceName string, namespace string) (*infrastructure.NodePortInfo, error) {
	c.logger.Info(ctx, "tekton nodeport client get service started",
		zap.String("service_name", serviceName),
		zap.String("namespace", namespace),
	)

	// Construct NodePort service name (pattern: {knative-service-name}-nodeport)
	nodeportServiceName := fmt.Sprintf("%s-nodeport", serviceName)

	// Get the Service
	svc, err := c.clientset.CoreV1().Services(namespace).Get(ctx, nodeportServiceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.logger.Info(ctx, "nodeport service not found",
				zap.String("service_name", nodeportServiceName),
			)
			return nil, containererrors.ErrNodePortNotFound
		}
		c.logger.Error(ctx, "failed to get nodeport service", zap.Error(err))
		return nil, containererrors.ErrKubernetesUnavailable
	}

	// Check if it's a NodePort service
	if svc.Spec.Type != corev1.ServiceTypeNodePort {
		c.logger.Error(ctx, "service is not a NodePort type",
			zap.String("type", string(svc.Spec.Type)),
		)
		return nil, containererrors.ErrNodePortNotFound
	}

	// Get NodePort number
	if len(svc.Spec.Ports) == 0 {
		c.logger.Error(ctx, "service has no ports")
		return nil, containererrors.ErrNodePortNotFound
	}

	nodePort := int(svc.Spec.Ports[0].NodePort)
	targetPort := int(svc.Spec.Ports[0].Port)

	// Get DNS name from ConfigMap
	host, err := c.getDNSName(ctx)
	if err != nil {
		c.logger.Warn(ctx, "failed to get DNS name from configmap, using empty", zap.Error(err))
		host = ""
	}

	// Parse timestamps from annotations
	createdAtStr := svc.Annotations["temporary-nodeport/created-at"]
	ttlSecondsStr := svc.Annotations["temporary-nodeport/ttl-seconds"]

	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	var expiresAt time.Time
	if ttlSecondsStr != "" {
		var ttlSeconds int
		if _, err := fmt.Sscanf(ttlSecondsStr, "%d", &ttlSeconds); err == nil {
			expiresAt = createdAt.Add(time.Duration(ttlSeconds) * time.Second)
		}
	}

	c.logger.Info(ctx, "tekton nodeport client get service completed",
		zap.String("host", host),
		zap.Int("port", nodePort),
	)

	return &infrastructure.NodePortInfo{
		ServiceName: nodeportServiceName,
		Namespace:   namespace,
		Host:        host,
		Port:        nodePort,
		TargetPort:  targetPort,
		Protocol:    "tcp",
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
		Status:      "active",
	}, nil
}

// getDNSName retrieves the DNS name from temporary-nodeport-config ConfigMap
func (c *tektonNodePortClient) getDNSName(ctx context.Context) (string, error) {
	configMap, err := c.clientset.CoreV1().ConfigMaps(c.configMapNS).Get(ctx, "temporary-nodeport-config", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	dnsName, exists := configMap.Data["dns_name"]
	if !exists {
		return "", fmt.Errorf("dns_name not found in configmap")
	}

	return dnsName, nil
}

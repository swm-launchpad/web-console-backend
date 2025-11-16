package health

import (
	"context"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KubernetesChecker checks the health of Kubernetes cluster
type KubernetesChecker struct {
	clientset *kubernetes.Clientset
}

// NewKubernetesChecker creates a new KubernetesChecker
func NewKubernetesChecker(clientset *kubernetes.Clientset) service.HealthChecker {
	return &KubernetesChecker{
		clientset: clientset,
	}
}

// ServiceName returns the service name
func (c *KubernetesChecker) ServiceName() value.ServiceName {
	return value.ServiceKubernetes
}

// Check performs the health check
func (c *KubernetesChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	start := time.Now()

	// Check Kubernetes API health
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get server version: %v", err)
		return model.NewStatusCheck(value.ServiceKubernetes, value.StatusDown, nil, &errorMsg), nil
	}

	// Check if at least one node is ready
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("failed to list nodes: %v", err)
		return model.NewStatusCheck(value.ServiceKubernetes, value.StatusDegraded, nil, &errorMsg), nil
	}

	if len(nodes.Items) == 0 {
		errorMsg := "no nodes found in cluster"
		return model.NewStatusCheck(value.ServiceKubernetes, value.StatusDown, nil, &errorMsg), nil
	}

	readyNodes := 0
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				readyNodes++
				break
			}
		}
	}

	responseTime := uint32(time.Since(start).Milliseconds())

	if readyNodes == 0 {
		errorMsg := "no ready nodes in cluster"
		return model.NewStatusCheck(value.ServiceKubernetes, value.StatusDown, &responseTime, &errorMsg), nil
	}

	if readyNodes < len(nodes.Items) {
		errorMsg := fmt.Sprintf("only %d/%d nodes are ready", readyNodes, len(nodes.Items))
		return model.NewStatusCheck(value.ServiceKubernetes, value.StatusDegraded, &responseTime, &errorMsg), nil
	}

	return model.NewStatusCheck(value.ServiceKubernetes, value.StatusOperational, &responseTime, nil), nil
}

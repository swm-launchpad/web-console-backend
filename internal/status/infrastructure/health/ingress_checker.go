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

// IngressChecker checks the health of Ingress Controller
type IngressChecker struct {
	clientset *kubernetes.Clientset
	namespace string
}

// NewIngressChecker creates a new IngressChecker
func NewIngressChecker(clientset *kubernetes.Clientset, namespace string) service.HealthChecker {
	return &IngressChecker{
		clientset: clientset,
		namespace: namespace,
	}
}

// ServiceName returns the service name
func (c *IngressChecker) ServiceName() value.ServiceName {
	return value.ServiceIngressService
}

// Check performs the health check
func (c *IngressChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	start := time.Now()

	// Check if ingress controller pods are running
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=ingress-nginx",
	})
	if err != nil {
		errorMsg := fmt.Sprintf("failed to list ingress controller pods: %v", err)
		return model.NewStatusCheck(value.ServiceIngressService, value.StatusDown, nil, &errorMsg), nil
	}

	if len(pods.Items) == 0 {
		errorMsg := "no ingress controller pods found"
		return model.NewStatusCheck(value.ServiceIngressService, value.StatusDown, nil, &errorMsg), nil
	}

	runningPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			allContainersReady := true
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if !containerStatus.Ready {
					allContainersReady = false
					break
				}
			}
			if allContainersReady {
				runningPods++
			}
		}
	}

	responseTime := uint32(time.Since(start).Milliseconds())

	if runningPods == 0 {
		errorMsg := "no ingress controller pods are running"
		return model.NewStatusCheck(value.ServiceIngressService, value.StatusDown, &responseTime, &errorMsg), nil
	}

	if runningPods < len(pods.Items) {
		errorMsg := fmt.Sprintf("only %d/%d ingress controller pods are running", runningPods, len(pods.Items))
		return model.NewStatusCheck(value.ServiceIngressService, value.StatusDegraded, &responseTime, &errorMsg), nil
	}

	return model.NewStatusCheck(value.ServiceIngressService, value.StatusOperational, &responseTime, nil), nil
}

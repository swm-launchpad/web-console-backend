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

// TektonChecker checks the health of Tekton by checking its pods
type TektonChecker struct {
	clientset *kubernetes.Clientset
	namespace string
}

// NewTektonChecker creates a new TektonChecker
func NewTektonChecker(clientset *kubernetes.Clientset, namespace string) service.HealthChecker {
	return &TektonChecker{
		clientset: clientset,
		namespace: namespace,
	}
}

// ServiceName returns the service name
func (c *TektonChecker) ServiceName() value.ServiceName {
	return value.ServiceTekton
}

// Check performs the health check
func (c *TektonChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	start := time.Now()

	// Check if Tekton controller and webhook pods are running
	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=tekton-pipelines",
	})
	if err != nil {
		errorMsg := fmt.Sprintf("failed to list Tekton pods: %v", err)
		return model.NewStatusCheck(value.ServiceTekton, value.StatusDown, nil, &errorMsg), nil
	}

	if len(pods.Items) == 0 {
		errorMsg := "no Tekton pods found"
		return model.NewStatusCheck(value.ServiceTekton, value.StatusDown, nil, &errorMsg), nil
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
		errorMsg := "no Tekton pods are running"
		return model.NewStatusCheck(value.ServiceTekton, value.StatusDown, &responseTime, &errorMsg), nil
	}

	if runningPods < len(pods.Items) {
		errorMsg := fmt.Sprintf("only %d/%d Tekton pods are running", runningPods, len(pods.Items))
		return model.NewStatusCheck(value.ServiceTekton, value.StatusDegraded, &responseTime, &errorMsg), nil
	}

	return model.NewStatusCheck(value.ServiceTekton, value.StatusOperational, &responseTime, nil), nil
}

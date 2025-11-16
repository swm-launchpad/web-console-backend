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

// NFSChecker checks the health of NFS by checking PVC status
type NFSChecker struct {
	clientset *kubernetes.Clientset
	namespace string
	pvcName   string
}

// NewNFSChecker creates a new NFSChecker
func NewNFSChecker(clientset *kubernetes.Clientset, namespace, pvcName string) service.HealthChecker {
	return &NFSChecker{
		clientset: clientset,
		namespace: namespace,
		pvcName:   pvcName,
	}
}

// ServiceName returns the service name
func (c *NFSChecker) ServiceName() value.ServiceName {
	return value.ServiceNFS
}

// Check performs the health check
func (c *NFSChecker) Check(ctx context.Context) (*model.StatusCheck, error) {
	start := time.Now()

	// Check PVC status
	pvc, err := c.clientset.CoreV1().PersistentVolumeClaims(c.namespace).Get(ctx, c.pvcName, metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get PVC: %v", err)
		return model.NewStatusCheck(value.ServiceNFS, value.StatusDown, nil, &errorMsg), nil
	}

	responseTime := uint32(time.Since(start).Milliseconds())

	if pvc.Status.Phase != "Bound" {
		errorMsg := fmt.Sprintf("PVC is not bound: %s", pvc.Status.Phase)
		return model.NewStatusCheck(value.ServiceNFS, value.StatusDown, &responseTime, &errorMsg), nil
	}

	// Check if PV exists and is available
	if pvc.Spec.VolumeName == "" {
		errorMsg := "PVC has no volume name"
		return model.NewStatusCheck(value.ServiceNFS, value.StatusDegraded, &responseTime, &errorMsg), nil
	}

	pv, err := c.clientset.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get PV: %v", err)
		return model.NewStatusCheck(value.ServiceNFS, value.StatusDegraded, &responseTime, &errorMsg), nil
	}

	if pv.Spec.NFS == nil {
		errorMsg := "PV is not an NFS volume"
		return model.NewStatusCheck(value.ServiceNFS, value.StatusDegraded, &responseTime, &errorMsg), nil
	}

	return model.NewStatusCheck(value.ServiceNFS, value.StatusOperational, &responseTime, nil), nil
}

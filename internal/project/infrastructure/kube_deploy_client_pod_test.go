package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetProjectPodStatus_MultiplePods tests that when multiple pods exist,
// the most recently created pod is selected (as happens during rolling updates or scale-up)
func TestGetProjectPodStatus_MultiplePods(t *testing.T) {
	// Create fake clientset
	fakeClient := fake.NewSimpleClientset()

	ctx := context.Background()
	projectID := uint(123)
	namespace := "application"

	// Create three pods with different creation timestamps
	oldTime := time.Now().Add(-10 * time.Minute)
	middleTime := time.Now().Add(-5 * time.Minute)
	newTime := time.Now().Add(-1 * time.Minute)

	// Old pod (should not be selected)
	oldPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-123-old-abc",
			Namespace: namespace,
			Labels: map[string]string{
				"project-id": "123",
			},
			CreationTimestamp: metav1.Time{Time: oldTime},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true},
				{Name: "sidecar", Ready: true},
			},
		},
	}

	// Middle pod (should not be selected)
	middlePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-123-middle-def",
			Namespace: namespace,
			Labels: map[string]string{
				"project-id": "123",
			},
			CreationTimestamp: metav1.Time{Time: middleTime},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true},
				{Name: "sidecar", Ready: false},
			},
		},
	}

	// New pod (should be selected - most recent)
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-123-new-ghi",
			Namespace: namespace,
			Labels: map[string]string{
				"project-id": "123",
			},
			CreationTimestamp: metav1.Time{Time: newTime},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: false},
				{Name: "sidecar", Ready: false},
			},
		},
	}

	// Create pods in random order
	_, err := fakeClient.CoreV1().Pods(namespace).Create(ctx, middlePod, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = fakeClient.CoreV1().Pods(namespace).Create(ctx, oldPod, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = fakeClient.CoreV1().Pods(namespace).Create(ctx, newPod, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create kubeClient with fake clientset
	client := &kubeClient{
		clientset: fakeClient,
		logger:    logger.NewForTest(),
	}

	// Set environment variable for application namespace
	t.Setenv("KUBE_APPLICATION_NAMESPACE", namespace)

	// Execute GetProjectPodStatus
	podStatus, err := client.GetProjectPodStatus(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, podStatus)
	assert.True(t, podStatus.Exists)

	// Should select the most recent pod (newPod)
	assert.Equal(t, string(corev1.PodPending), podStatus.Phase)
	assert.Equal(t, 0, podStatus.ReadyContainers)
	assert.Equal(t, 2, podStatus.TotalContainers)
}

// TestGetProjectPodStatus_SinglePod tests normal case with single pod
func TestGetProjectPodStatus_SinglePod(t *testing.T) {
	// Create fake clientset
	fakeClient := fake.NewSimpleClientset()

	ctx := context.Background()
	projectID := uint(456)
	namespace := "application"

	// Create single pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-456-abc",
			Namespace: namespace,
			Labels: map[string]string{
				"project-id": "456",
			},
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true},
			},
		},
	}

	_, err := fakeClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Create kubeClient with fake clientset
	client := &kubeClient{
		clientset: fakeClient,
		logger:    logger.NewForTest(),
	}

	// Set environment variable for application namespace
	t.Setenv("KUBE_APPLICATION_NAMESPACE", namespace)

	// Execute GetProjectPodStatus
	podStatus, err := client.GetProjectPodStatus(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, podStatus)
	assert.True(t, podStatus.Exists)
	assert.Equal(t, string(corev1.PodRunning), podStatus.Phase)
	assert.Equal(t, 1, podStatus.ReadyContainers)
	assert.Equal(t, 1, podStatus.TotalContainers)
}

// TestGetProjectPodStatus_NoPods tests case when no pods exist
func TestGetProjectPodStatus_NoPods(t *testing.T) {
	// Create fake clientset (empty)
	fakeClient := fake.NewSimpleClientset()

	ctx := context.Background()
	projectID := uint(789)
	namespace := "application"

	// Create kubeClient with fake clientset
	client := &kubeClient{
		clientset: fakeClient,
		logger:    logger.NewForTest(),
	}

	// Set environment variable for application namespace
	t.Setenv("KUBE_APPLICATION_NAMESPACE", namespace)

	// Execute GetProjectPodStatus
	podStatus, err := client.GetProjectPodStatus(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, podStatus)
	assert.False(t, podStatus.Exists)
	assert.Equal(t, "", podStatus.Phase)
	assert.Equal(t, 0, podStatus.ReadyContainers)
	assert.Equal(t, 0, podStatus.TotalContainers)
}

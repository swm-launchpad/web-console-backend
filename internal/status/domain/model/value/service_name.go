package value

import "fmt"

// ServiceName represents the identifier of a monitored service
type ServiceName string

const (
	ServiceAPIServer      ServiceName = "api_server"
	ServiceWebConsole     ServiceName = "web_console"
	ServiceMySQL          ServiceName = "mysql"
	ServiceTekton         ServiceName = "tekton"
	ServiceRegistry       ServiceName = "container_registry"
	ServiceKubernetes     ServiceName = "kubernetes"
	ServiceNFS            ServiceName = "nfs"
	ServiceLoki           ServiceName = "loki"
	ServiceIngressService ServiceName = "ingress_controller"
)

// AllServiceNames returns all valid service names
func AllServiceNames() []ServiceName {
	return []ServiceName{
		ServiceAPIServer,
		ServiceWebConsole,
		ServiceMySQL,
		ServiceTekton,
		ServiceRegistry,
		ServiceKubernetes,
		ServiceNFS,
		ServiceLoki,
		ServiceIngressService,
	}
}

// IsValid checks if the service name is valid
func (s ServiceName) IsValid() bool {
	for _, valid := range AllServiceNames() {
		if s == valid {
			return true
		}
	}
	return false
}

// String returns the string representation
func (s ServiceName) String() string {
	return string(s)
}

// DisplayName returns the user-friendly display name
func (s ServiceName) DisplayName() string {
	displayNames := map[ServiceName]string{
		ServiceAPIServer:      "API Server",
		ServiceWebConsole:     "Web Console",
		ServiceMySQL:          "Database",
		ServiceTekton:         "Build System",
		ServiceRegistry:       "Container Registry",
		ServiceKubernetes:     "Container Platform",
		ServiceNFS:            "Storage Service",
		ServiceLoki:           "Logging System",
		ServiceIngressService: "Ingress Controller",
	}
	if name, ok := displayNames[s]; ok {
		return name
	}
	return string(s)
}

// Description returns the service description
func (s ServiceName) Description() string {
	descriptions := map[ServiceName]string{
		ServiceAPIServer:      "Go/Gin backend API server",
		ServiceWebConsole:     "React/Vite frontend application",
		ServiceMySQL:          "MySQL database server",
		ServiceTekton:         "Tekton CI/CD build system",
		ServiceRegistry:       "AWS ECR container registry",
		ServiceKubernetes:     "Kubernetes container platform",
		ServiceNFS:            "NFS storage service",
		ServiceLoki:           "Loki log aggregation system",
		ServiceIngressService: "NGINX ingress controller",
	}
	if desc, ok := descriptions[s]; ok {
		return desc
	}
	return ""
}

// NewServiceName creates and validates a ServiceName
func NewServiceName(name string) (ServiceName, error) {
	sn := ServiceName(name)
	if !sn.IsValid() {
		return "", fmt.Errorf("invalid service name: %s", name)
	}
	return sn, nil
}

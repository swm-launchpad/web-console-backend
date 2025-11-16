package model

import (
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
)

// ServiceInfo contains metadata and configuration for a monitored service
type ServiceInfo struct {
	Name         value.ServiceName
	Category     value.ServiceCategory
	DisplayName  string
	Description  string
	Icon         string // emoji or icon identifier
	CheckTimeout time.Duration
}

// NewServiceInfo creates a new ServiceInfo
func NewServiceInfo(name value.ServiceName) *ServiceInfo {
	return &ServiceInfo{
		Name:         name,
		Category:     value.GetCategoryForService(name),
		DisplayName:  name.DisplayName(),
		Description:  name.Description(),
		Icon:         getServiceIcon(name),
		CheckTimeout: 5 * time.Second, // default timeout
	}
}

// getServiceIcon returns the icon for a service
func getServiceIcon(name value.ServiceName) string {
	icons := map[value.ServiceName]string{
		value.ServiceAPIServer:      "🚀",
		value.ServiceWebConsole:     "💻",
		value.ServiceMySQL:          "🗄️",
		value.ServiceTekton:         "🔨",
		value.ServiceRegistry:       "📦",
		value.ServiceKubernetes:     "☸️",
		value.ServiceNFS:            "💾",
		value.ServiceLoki:           "📊",
		value.ServiceIngressService: "🌐",
	}
	if icon, ok := icons[name]; ok {
		return icon
	}
	return "⚙️"
}

// AllServiceInfos returns ServiceInfo for all monitored services
func AllServiceInfos() []*ServiceInfo {
	services := value.AllServiceNames()
	infos := make([]*ServiceInfo, len(services))
	for i, name := range services {
		infos[i] = NewServiceInfo(name)
	}
	return infos
}

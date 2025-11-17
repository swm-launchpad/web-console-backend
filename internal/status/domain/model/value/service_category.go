package value

import "fmt"

// ServiceCategory represents the category of a service
type ServiceCategory string

const (
	CategoryCore           ServiceCategory = "core"
	CategoryBuildDeploy    ServiceCategory = "build_deploy"
	CategoryInfrastructure ServiceCategory = "infrastructure"
)

// IsValid checks if the category is valid
func (c ServiceCategory) IsValid() bool {
	switch c {
	case CategoryCore, CategoryBuildDeploy, CategoryInfrastructure:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (c ServiceCategory) String() string {
	return string(c)
}

// DisplayName returns the user-friendly display name
func (c ServiceCategory) DisplayName() string {
	switch c {
	case CategoryCore:
		return "Core Services"
	case CategoryBuildDeploy:
		return "Build & Deploy"
	case CategoryInfrastructure:
		return "Infrastructure"
	default:
		return string(c)
	}
}

// GetCategoryForService returns the category for a given service
func GetCategoryForService(service ServiceName) ServiceCategory {
	categoryMap := map[ServiceName]ServiceCategory{
		ServiceAPIServer:      CategoryCore,
		ServiceWebConsole:     CategoryCore,
		ServiceMySQL:          CategoryCore,
		ServiceTekton:         CategoryBuildDeploy,
		ServiceRegistry:       CategoryBuildDeploy,
		ServiceLoki:           CategoryBuildDeploy,
		ServiceKubernetes:     CategoryInfrastructure,
		ServiceNFS:            CategoryInfrastructure,
		ServiceIngressService: CategoryInfrastructure,
	}
	if category, ok := categoryMap[service]; ok {
		return category
	}
	return ""
}

// NewServiceCategory creates and validates a ServiceCategory
func NewServiceCategory(category string) (ServiceCategory, error) {
	c := ServiceCategory(category)
	if !c.IsValid() {
		return "", fmt.Errorf("invalid service category: %s", category)
	}
	return c, nil
}

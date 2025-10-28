package service

// BuildParametersSnapshot represents a snapshot of build-affecting parameters
// This is used to detect if any build parameters changed during the build process
type BuildParametersSnapshot struct {
	GitRepositoryURL string
	GitBranch        string
	GitDirectoryPath *string
	TemplateID       *uint
	TemplateConfig   map[string]interface{}
	BuildVars        map[string]string
	InstallationID   *int64
}

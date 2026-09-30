package domain

type BootstrapResponse struct {
	Features           map[string]bool `json:"features"`
	MinRequiredVersion string          `json:"min_required_version"`
}

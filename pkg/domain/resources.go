package domain

type UnstructuredResource struct {
	Name string `yaml:"name,omitempty"`
	// Path represents e.g. gvk in kubernetes
	Path    string                 `json:"path,omitempty"`
	Content map[string]interface{} `yaml:"content,omitempty"`
}

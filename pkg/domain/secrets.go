package domain

type SecretYaml struct {
	ApiVersion string             `yaml:"apiVersion,omitempty"`
	Kind       string             `yaml:"kind,omitempty"`
	SecretType string             `yaml:"type,omitempty"`
	Data       map[string]string  `yaml:"data,omitempty"`
	Metadata   SecretYamlMetaData `yaml:"metadata,omitempty"`
}

type SecretYamlMetaData struct {
	Name              string            `yaml:"name,omitempty"`
	Namespace         string            `yaml:"namespace,omitempty"`
	UID               string            `yaml:"uid,omitempty"`
	CreationTimestamp string            `yaml:"creationTimestamp,omitempty"`
	Labels            map[string]string `yaml:"labels,omitempty"`
}

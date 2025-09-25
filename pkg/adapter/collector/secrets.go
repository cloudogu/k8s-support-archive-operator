package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const censoredValue = "***"
const labelSelector = "app=ces"
const labelConfigTypeKey = "k8s.cloudogu.com/type"
const labelSensitiveConfigType = "sensitive-config"
const configYamlKey = "config.yaml"

type SecretCollector struct {
	coreV1Interface coreV1Interface
}

func NewSecretCollector(coreV1Interface coreV1Interface) *SecretCollector {
	return &SecretCollector{coreV1Interface: coreV1Interface}
}

func (sc *SecretCollector) Name() string {
	return string(domain.CollectorTypeSecret)
}

func (sc *SecretCollector) Collect(ctx context.Context, namespace string, _, _ time.Time, resultChan chan<- *domain.SecretYaml) error {
	defer close(resultChan)

	logger := log.FromContext(ctx).WithName("SecretCollector.Collect")
	list, err := sc.coreV1Interface.Secrets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("error listing secrets: %w", err)
	}

	if len(list.Items) == 0 {
		logger.Info("Secret list is empty")
		return nil
	}

	for _, secret := range list.Items {
		censored := sc.censorSecret(secret)
		logger.Info(fmt.Sprintf("censored secret %q and write it into channel", secret.Name))
		writeSaveToChannel(ctx, censored, resultChan)
	}

	return nil
}

func (sc *SecretCollector) censorSecret(secret v1.Secret) *domain.SecretYaml {
	censored := &domain.SecretYaml{
		ApiVersion: secret.APIVersion,
		Kind:       secret.Kind,
		SecretType: string(secret.Type),
		Data:       map[string]string{},
		Metadata: domain.SecretYamlMetaData{
			Name:              secret.Name,
			Namespace:         secret.Namespace,
			UID:               string(secret.UID),
			CreationTimestamp: secret.CreationTimestamp.Format(time.RFC3339),
			Labels:            secret.Labels,
		},
	}

	if secret.Labels[labelConfigTypeKey] == labelSensitiveConfigType {
		var yamlNode yaml.Node
		if err := yaml.Unmarshal(secret.Data[configYamlKey], &yamlNode); err == nil {
			censorYaml(&yamlNode)
			encoded, err := yaml.Marshal(&yamlNode)
			if err == nil {
				censored.Data[configYamlKey] = string(encoded)
				return censored
			}
		}
	}

	for key := range secret.Data {
		censored.Data[key] = censoredValue
	}
	return censored
}

func censorYaml(node *yaml.Node) {
	switch node.Kind {
	case yaml.DocumentNode:
		censorYAMLNode(node)
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			valueNode := node.Content[i+1]
			censorYaml(valueNode)
		}
	case yaml.SequenceNode:
		censorYAMLNode(node)
	case yaml.ScalarNode:
		node.Value = censoredValue
	}
}

func censorYAMLNode(node *yaml.Node) {
	for _, n := range node.Content {
		censorYaml(n)
	}
}

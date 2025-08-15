package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const censoredValue = "***"
const labelSelector = "app=ces"

type SecretCollector struct {
	coreV1Interface coreV1Interface
}

func NewSecretCollector(coreV1Interface coreV1Interface) *SecretCollector {
	return &SecretCollector{coreV1Interface: coreV1Interface}
}

func (sc *SecretCollector) Name() string {
	return string(domain.CollectorTypSecret)
}

func (sc *SecretCollector) Collect(ctx context.Context, namespace string, _, _ time.Time, resultChan chan<- *domain.SecretYaml) error {
	logger := log.FromContext(ctx).WithName("SecretCollector.Collect")
	list, err := sc.coreV1Interface.Secrets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("error listing secrets: %w", err)
	}

	if len(list.Items) == 0 {
		logger.Info("Secret list is empty")
		close(resultChan)
		return nil
	}

	for _, secret := range list.Items {
		censored := sc.censorSecret(secret)
		logger.Info(fmt.Sprintf("censored secret: %s and write it into channel", secret.Name))
		writeSaveToChannel(ctx, censored, resultChan)
	}

	close(resultChan)
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
			CreationTimestamp: secret.CreationTimestamp.String(),
			Labels:            secret.Labels,
		},
	}

	for key, val := range secret.Data {
		// Try json parse
		var parsed interface{}
		if err := json.Unmarshal(val, &parsed); err == nil {
			censoredJSON := censorJsonSecret(parsed)
			newVal, err := json.Marshal(censoredJSON)
			if err == nil {
				censored.Data[key] = string(newVal)
			}
			continue
		}

		// Try yaml parse
		var yamlNode yaml.Node
		if err := yaml.Unmarshal(val, &yamlNode); err == nil {
			if isSingleScalarNode(&yamlNode) {
				censored.Data[key] = censoredValue
				continue
			}
			censorYaml(&yamlNode)
			encoded, err := yaml.Marshal(&yamlNode)
			if err == nil {
				censored.Data[key] = string(encoded)
			}
			continue
		}
	}
	return censored
}

func censorJsonSecret(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for k := range v {
			v[k] = censorJsonSecret(v[k])
		}
		return v
	case []interface{}:
		for i := range v {
			v[i] = censorJsonSecret(v[i])
		}
		return v
	default:
		return "***"
	}
}

func censorYaml(node *yaml.Node) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, n := range node.Content {
			censorYaml(n)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			valueNode := node.Content[i+1]
			censorYaml(valueNode)
		}
	case yaml.SequenceNode:
		for _, n := range node.Content {
			censorYaml(n)
		}
	case yaml.ScalarNode:
		node.Value = censoredValue
	}
}

func isSingleScalarNode(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode || (node.Kind == yaml.DocumentNode && len(node.Content) == 1 && node.Content[0].Kind == yaml.ScalarNode)
}

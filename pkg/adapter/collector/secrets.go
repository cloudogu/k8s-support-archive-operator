package collector

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

const censoredValue = "***"

type SecretCollector struct {
	coreV1Interface coreV1Interface
}

func NewSecretCollector(coreV1Interface coreV1Interface) *SecretCollector {
	return &SecretCollector{coreV1Interface: coreV1Interface}
}

func (sc *SecretCollector) Name() string {
	return string(domain.CollectorTypSecret)
}

func (sc *SecretCollector) Collect(ctx context.Context, namespace string, _, _ time.Time, resultChan chan<- *v1.SecretList) error {
	logger := log.FromContext(ctx).WithName("SecretCollector.Collect")
	labelSelector := "app=ces"
	list, err := sc.coreV1Interface.Secrets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("error listing secrets: %w", err)
	}
	logger.Info("Fetched all Secrets")

	if len(list.Items) == 0 {
		logger.Info("Secret list is empty")
		close(resultChan)
		return nil
	}

	secrets := make([]v1.Secret, 0)

	for _, secret := range list.Items {
		censored := sc.censorSecret(secret)
		for key, val := range censored.Data {
			logger.Info(fmt.Sprintf("Censored key=%s, val=%s", key, val))
		}
		secrets = append(secrets, *censored)
		logger.Info(fmt.Sprintf("censored secret: %s", secret.Name))
	}
	logger.Info("censored all secrets")

	logger.Info("write all secrets into channel")
	writeSaveToChannel(ctx, &v1.SecretList{Items: secrets}, resultChan)
	close(resultChan)

	logger.Info("secret channel is closed")
	return nil
}

func (sc *SecretCollector) censorSecret(secret v1.Secret) *v1.Secret {
	censored := secret.DeepCopy()
	for key, val := range secret.Data {
		// Try json parse
		var parsed interface{}
		err := json.Unmarshal(val, &parsed)
		if err == nil {
			censoredJSON := censorJsonSecret(parsed)
			newVal, err := json.Marshal(censoredJSON)
			if err == nil {
				censored.Data[key] = newVal
			}
			continue
		}

		// Try yaml parse
		var yamlNode yaml.Node
		if err := yaml.Unmarshal(val, &yamlNode); err == nil {
			censorYaml(&yamlNode)
			encoded, err := yaml.Marshal(&yamlNode)
			if err != nil {
				return nil
			}
			censored.Data[key] = encoded
			continue
		}

		// Censor key with default censorValue
		censored.Data[key] = []byte(censoredValue)
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

package collector

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

const (
	listVerb  = "list"
	coreGroup = "core"
)

type gvkMatcher schema.GroupVersionKind

// Matches checks if the fields of the supplied schema.GroupVersionKind equal those of the gvkMatcher.
// Particular fields can be ignored by using the star-notation (*) in the matcher.
func (m gvkMatcher) Matches(gvk schema.GroupVersionKind) bool {
	return (gvk.Group == m.Group || m.Group == "*") &&
		(gvk.Version == m.Version || m.Version == "*") &&
		(gvk.Kind == m.Kind || m.Kind == "*")
}

type SystemStateCollector struct {
	client                k8sClient
	discoveryClient       discoveryInterface
	resourceLabelSelector *metav1.LabelSelector
	excludedGVKs          []gvkMatcher
}

// NewSystemStateCollector creates a SystemStateCollector.
func NewSystemStateCollector(genericClient k8sClient, discoveryClient discoveryInterface, labelSelectorsYaml string, gvkExclusionsYaml string) (*SystemStateCollector, error) {
	var matchLabels = map[string]string{}
	err := yaml.Unmarshal([]byte(labelSelectorsYaml), matchLabels)
	if err != nil {
		return nil, err
	}

	gvkExclusions := make([]gvkMatcher, 0)
	err = yaml.Unmarshal([]byte(gvkExclusionsYaml), &gvkExclusions)
	if err != nil {
		return nil, err
	}

	return &SystemStateCollector{
		client:                genericClient,
		discoveryClient:       discoveryClient,
		resourceLabelSelector: &metav1.LabelSelector{MatchLabels: matchLabels},
		excludedGVKs:          gvkExclusions,
	}, nil
}

func (rc *SystemStateCollector) Name() string {
	return string(domain.CollectorTypeSystemState)
}

func (rc *SystemStateCollector) Collect(ctx context.Context, namespace string, _, _ time.Time, resultChan chan<- *domain.UnstructuredResource) error {
	resourceKindLists, err := rc.discoveryClient.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("failed to get resource kind lists from server: %w", err)
	}

	selector, err := metav1.LabelSelectorAsSelector(rc.resourceLabelSelector)
	if err != nil {
		return fmt.Errorf("failed to create selector from given label selector %s: %w", rc.resourceLabelSelector, err)
	}

	var errs []error
	var resources []*unstructured.Unstructured
	for _, resourceKindList := range resourceKindLists {
		resourcesOfKind, listErrs := rc.listApiResourcesByLabelSelector(ctx, namespace, resourceKindList, selector, rc.excludedGVKs)
		resources = append(resources, resourcesOfKind...)
		errs = append(errs, listErrs...)
	}

	if len(errs) != 0 {
		return fmt.Errorf("failed to list api resources with label selector %q: %w", selector, errors.Join(errs...))
	}

	for _, resource := range resources {
		gvk := resource.GroupVersionKind()
		group := gvk.Group
		if group == "" {
			group = coreGroup
		}
		resultChan <- &domain.UnstructuredResource{
			Name:    resource.GetName(),
			Path:    filepath.Join(group, gvk.Version, gvk.Kind),
			Content: resource.Object,
		}
	}

	close(resultChan)
	return nil
}

func (rc *SystemStateCollector) listApiResourcesByLabelSelector(ctx context.Context, namespace string, list *metav1.APIResourceList, selector labels.Selector, excludedGVKs []gvkMatcher) ([]*unstructured.Unstructured, []error) {
	if len(list.APIResources) == 0 {
		return nil, nil
	}

	gv, err := schema.ParseGroupVersion(list.GroupVersion)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to list api resources with group version %q: %w", list.GroupVersion, err)}
	}

	var errs []error
	var resources []*unstructured.Unstructured
	for _, resource := range list.APIResources {
		if len(resource.Verbs) != 0 && slices.Contains(resource.Verbs, listVerb) {
			resource.Group = gv.Group
			resource.Version = gv.Version

			resourcesByLabelSelector, listErr := rc.listByLabelSelector(ctx, namespace, resource, selector, excludedGVKs)
			if listErr != nil {
				errs = append(errs, listErr)
			} else {
				resources = append(resources, resourcesByLabelSelector...)
			}
		}
	}

	return resources, errs
}

func (rc *SystemStateCollector) listByLabelSelector(ctx context.Context, namespace string, resource metav1.APIResource, labelSelector labels.Selector, excludedGVKs []gvkMatcher) ([]*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)

	gvk := groupVersionKind(resource)
	for _, matcher := range excludedGVKs {
		if matcher.Matches(gvk) {
			logger.Info(fmt.Sprintf("skipping resource %s as it is excluded", gvk))
			return nil, nil
		}
	}
	listOptions := client.ListOptions{LabelSelector: &client.MatchingLabelsSelector{Selector: labelSelector}}
	if resource.Namespaced {
		listOptions.Namespace = namespace
	}

	objectList := &unstructured.UnstructuredList{}
	objectList.SetGroupVersionKind(gvk)
	err := rc.client.List(ctx, objectList, &listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in %s: %w", gvk, err)
	}

	return sliceToPointers(objectList.Items), nil
}

func groupVersionKind(resource metav1.APIResource) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   resource.Group,
		Version: resource.Version,
		Kind:    resource.Kind,
	}
}

func sliceToPointers[T any](raw []T) []*T {
	pointers := make([]*T, len(raw))
	for i, t := range raw {
		pointers[i] = &t
	}
	return pointers
}

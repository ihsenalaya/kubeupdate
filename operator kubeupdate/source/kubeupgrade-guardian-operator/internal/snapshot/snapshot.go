/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package snapshot is the single read point between the operator and the cluster.
// It performs exactly one paginated List per resource type through a direct
// (uncached) reader, so an assessment observes one consistent cluster state and
// no informer is started for cluster-wide types.
package snapshot

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	policyv1 "k8s.io/api/policy/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

// listPageSize bounds the memory held while paging through a resource type.
const listPageSize = 500

// Resource names used in gap findings. Checkers use them to tell "the cluster has
// none of these" apart from "we were not allowed to look".
const (
	ResourceNamespaces         = "namespaces"
	ResourceDeployments        = "deployments"
	ResourceStatefulSets       = "statefulsets"
	ResourceDaemonSets         = "daemonsets"
	ResourcePDBs               = "poddisruptionbudgets"
	ResourceHPAs               = "horizontalpodautoscalers"
	ResourceCronJobs           = "cronjobs"
	ResourcePods               = "pods"
	ResourceNodes              = "nodes"
	ResourceServices           = "services"
	ResourceEndpointSlices     = "endpointslices"
	ResourceValidatingWebhooks = "validatingwebhookconfigurations"
	ResourceMutatingWebhooks   = "mutatingwebhookconfigurations"
	ResourceCRDs               = "customresourcedefinitions"
)

// ClusterSnapshot is the read-only view of the cluster an assessment runs against.
//
// Namespaced workload inventories (Deployments, StatefulSets, DaemonSets, PDBs,
// HPAs, CronJobs) are already reduced to the assessment scope. Cluster-level
// signals (Nodes, Pods, Services, EndpointSlices, webhook configurations, CRDs)
// are kept cluster-wide because the checks consuming them reason about the whole
// cluster: capacity headroom is a cluster property, and a webhook backend Service
// usually lives outside the assessed namespaces.
type ClusterSnapshot struct {
	TakenAt time.Time

	// Namespaces are the namespaces retained by the assessment scope.
	Namespaces []corev1.Namespace
	// AllNamespaces is every namespace observed, regardless of scope. Cluster-level
	// capability discovery (observability) must not be narrowed by the scope.
	AllNamespaces []corev1.Namespace

	Deployments        []appsv1.Deployment
	StatefulSets       []appsv1.StatefulSet
	DaemonSets         []appsv1.DaemonSet
	Pods               []corev1.Pod
	PDBs               []policyv1.PodDisruptionBudget
	Nodes              []corev1.Node
	Services           []corev1.Service
	EndpointSlices     []discoveryv1.EndpointSlice
	HPAs               []autoscalingv2.HorizontalPodAutoscaler
	CronJobs           []batchv1.CronJob
	ValidatingWebhooks []admissionv1.ValidatingWebhookConfiguration
	MutatingWebhooks   []admissionv1.MutatingWebhookConfiguration
	CRDNames           map[string]struct{}

	// Gaps holds the RBAC_ASSESSMENT_GAP findings raised while collecting. A denied
	// resource type yields one gap and an empty slice; collection keeps going.
	Gaps []upgradev1alpha1.Finding
}

// Collect reads every resource type the checkers need, once, through reader.
// An RBAC refusal on a type is recorded as a gap finding and never fails the
// assessment; any other error aborts collection.
func Collect(ctx context.Context, reader client.Reader, scope upgradev1alpha1.AssessmentScope) (*ClusterSnapshot, error) {
	snap := &ClusterSnapshot{
		TakenAt:  time.Now().UTC(),
		CRDNames: map[string]struct{}{},
	}

	allNamespaces, err := collect(ctx, reader, snap, ResourceNamespaces,
		func() *corev1.NamespaceList { return &corev1.NamespaceList{} },
		func(list *corev1.NamespaceList) []corev1.Namespace { return list.Items })
	if err != nil {
		return nil, err
	}
	snap.AllNamespaces = allNamespaces
	snap.Namespaces = scopedNamespaces(allNamespaces, scope)
	scoped := namespaceSet(snap.Namespaces)

	deployments, err := collect(ctx, reader, snap, ResourceDeployments,
		func() *appsv1.DeploymentList { return &appsv1.DeploymentList{} },
		func(list *appsv1.DeploymentList) []appsv1.Deployment { return list.Items })
	if err != nil {
		return nil, err
	}
	snap.Deployments = filterByNamespace(deployments, scoped, func(item appsv1.Deployment) string { return item.Namespace })

	statefulSets, err := collect(ctx, reader, snap, ResourceStatefulSets,
		func() *appsv1.StatefulSetList { return &appsv1.StatefulSetList{} },
		func(list *appsv1.StatefulSetList) []appsv1.StatefulSet { return list.Items })
	if err != nil {
		return nil, err
	}
	snap.StatefulSets = filterByNamespace(statefulSets, scoped, func(item appsv1.StatefulSet) string { return item.Namespace })

	daemonSets, err := collect(ctx, reader, snap, ResourceDaemonSets,
		func() *appsv1.DaemonSetList { return &appsv1.DaemonSetList{} },
		func(list *appsv1.DaemonSetList) []appsv1.DaemonSet { return list.Items })
	if err != nil {
		return nil, err
	}
	snap.DaemonSets = filterByNamespace(daemonSets, scoped, func(item appsv1.DaemonSet) string { return item.Namespace })

	pdbs, err := collect(ctx, reader, snap, ResourcePDBs,
		func() *policyv1.PodDisruptionBudgetList { return &policyv1.PodDisruptionBudgetList{} },
		func(list *policyv1.PodDisruptionBudgetList) []policyv1.PodDisruptionBudget { return list.Items })
	if err != nil {
		return nil, err
	}
	snap.PDBs = filterByNamespace(pdbs, scoped, func(item policyv1.PodDisruptionBudget) string { return item.Namespace })

	hpas, err := collect(ctx, reader, snap, ResourceHPAs,
		func() *autoscalingv2.HorizontalPodAutoscalerList { return &autoscalingv2.HorizontalPodAutoscalerList{} },
		func(list *autoscalingv2.HorizontalPodAutoscalerList) []autoscalingv2.HorizontalPodAutoscaler {
			return list.Items
		})
	if err != nil {
		return nil, err
	}
	snap.HPAs = filterByNamespace(hpas, scoped, func(item autoscalingv2.HorizontalPodAutoscaler) string { return item.Namespace })

	cronJobs, err := collect(ctx, reader, snap, ResourceCronJobs,
		func() *batchv1.CronJobList { return &batchv1.CronJobList{} },
		func(list *batchv1.CronJobList) []batchv1.CronJob { return list.Items })
	if err != nil {
		return nil, err
	}
	snap.CronJobs = filterByNamespace(cronJobs, scoped, func(item batchv1.CronJob) string { return item.Namespace })

	if snap.Pods, err = collect(ctx, reader, snap, ResourcePods,
		func() *corev1.PodList { return &corev1.PodList{} },
		func(list *corev1.PodList) []corev1.Pod { return list.Items }); err != nil {
		return nil, err
	}

	if snap.Nodes, err = collect(ctx, reader, snap, ResourceNodes,
		func() *corev1.NodeList { return &corev1.NodeList{} },
		func(list *corev1.NodeList) []corev1.Node { return list.Items }); err != nil {
		return nil, err
	}

	if snap.Services, err = collect(ctx, reader, snap, ResourceServices,
		func() *corev1.ServiceList { return &corev1.ServiceList{} },
		func(list *corev1.ServiceList) []corev1.Service { return list.Items }); err != nil {
		return nil, err
	}

	if snap.EndpointSlices, err = collect(ctx, reader, snap, ResourceEndpointSlices,
		func() *discoveryv1.EndpointSliceList { return &discoveryv1.EndpointSliceList{} },
		func(list *discoveryv1.EndpointSliceList) []discoveryv1.EndpointSlice { return list.Items }); err != nil {
		return nil, err
	}

	if snap.ValidatingWebhooks, err = collect(ctx, reader, snap, ResourceValidatingWebhooks,
		func() *admissionv1.ValidatingWebhookConfigurationList {
			return &admissionv1.ValidatingWebhookConfigurationList{}
		},
		func(list *admissionv1.ValidatingWebhookConfigurationList) []admissionv1.ValidatingWebhookConfiguration {
			return list.Items
		}); err != nil {
		return nil, err
	}

	if snap.MutatingWebhooks, err = collect(ctx, reader, snap, ResourceMutatingWebhooks,
		func() *admissionv1.MutatingWebhookConfigurationList {
			return &admissionv1.MutatingWebhookConfigurationList{}
		},
		func(list *admissionv1.MutatingWebhookConfigurationList) []admissionv1.MutatingWebhookConfiguration {
			return list.Items
		}); err != nil {
		return nil, err
	}

	crds, err := collect(ctx, reader, snap, ResourceCRDs,
		func() *apiextensionsv1.CustomResourceDefinitionList {
			return &apiextensionsv1.CustomResourceDefinitionList{}
		},
		func(list *apiextensionsv1.CustomResourceDefinitionList) []apiextensionsv1.CustomResourceDefinition {
			return list.Items
		})
	if err != nil {
		return nil, err
	}
	for _, crd := range crds {
		snap.CRDNames[crd.Name] = struct{}{}
	}

	sort.SliceStable(snap.Gaps, func(i, j int) bool { return snap.Gaps[i].ID < snap.Gaps[j].ID })
	return snap, nil
}

// Denied reports whether collecting a resource type was refused by RBAC. Checks
// that read an absence as a problem must not fire when the absence only means the
// operator was not allowed to look.
func (s *ClusterSnapshot) Denied(resource string) bool {
	for _, gap := range s.Gaps {
		for _, evidence := range gap.Evidence {
			if evidence.Observed["resource"] == resource {
				return true
			}
		}
	}
	return false
}

// collect pages through one resource type. A forbidden response is downgraded to
// a gap finding so a partially authorized service account still produces a report.
func collect[L client.ObjectList, T any](
	ctx context.Context,
	reader client.Reader,
	snap *ClusterSnapshot,
	resource string,
	newList func() L,
	itemsOf func(L) []T,
) ([]T, error) {
	var (
		out           []T
		continueToken string
	)
	for {
		list := newList()
		opts := []client.ListOption{client.Limit(listPageSize)}
		if continueToken != "" {
			opts = append(opts, client.Continue(continueToken))
		}
		if err := reader.List(ctx, list, opts...); err != nil {
			if apierrors.IsForbidden(err) {
				snap.Gaps = append(snap.Gaps, gapFinding(resource, err))
				return nil, nil
			}
			return nil, fmt.Errorf("listing %s: %w", resource, err)
		}
		out = append(out, itemsOf(list)...)
		continueToken = list.GetContinue()
		if continueToken == "" {
			return out, nil
		}
	}
}

// scopedNamespaces mirrors the include/exclude semantics the checkers used before
// the snapshot existed: an explicit include list wins, exclude always subtracts.
// Included namespaces that were not observed are kept as name-only entries so a
// scoped assessment still reports on them.
func scopedNamespaces(all []corev1.Namespace, scope upgradev1alpha1.AssessmentScope) []corev1.Namespace {
	include := stringSet(scope.Namespaces.Include)
	exclude := stringSet(scope.Namespaces.Exclude)

	var out []corev1.Namespace
	if len(include) > 0 {
		observed := make(map[string]corev1.Namespace, len(all))
		for _, namespace := range all {
			observed[namespace.Name] = namespace
		}
		out = make([]corev1.Namespace, 0, len(include))
		for name := range include {
			if _, blocked := exclude[name]; blocked {
				continue
			}
			if namespace, ok := observed[name]; ok {
				out = append(out, namespace)
				continue
			}
			out = append(out, corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})
		}
	} else {
		out = make([]corev1.Namespace, 0, len(all))
		for _, namespace := range all {
			if _, blocked := exclude[namespace.Name]; blocked {
				continue
			}
			out = append(out, namespace)
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func filterByNamespace[T any](items []T, scoped map[string]struct{}, namespaceOf func(T) string) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := scoped[namespaceOf(item)]; ok {
			out = append(out, item)
		}
	}
	return out
}

func namespaceSet(namespaces []corev1.Namespace) map[string]struct{} {
	set := make(map[string]struct{}, len(namespaces))
	for _, namespace := range namespaces {
		set[namespace.Name] = struct{}{}
	}
	return set
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	return set
}

func gapFinding(resource string, err error) upgradev1alpha1.Finding {
	id := gapID(resource)
	return upgradev1alpha1.Finding{
		ID:       id,
		Type:     upgradev1alpha1.FindingTypeRBACAssessmentGap,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "RBAC",
		Message:  fmt.Sprintf("Cluster snapshot could not read %s because Kubernetes RBAC denied access: %v", resource, err),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          id + "_EVIDENCE",
			Description: "RBAC denied a read operation required for assessment.",
			Observed: map[string]string{
				"resource": resource,
				"error":    err.Error(),
			},
		}},
		Recommendation: "Grant read-only permissions for this resource type, then rerun the assessment.",
	}
}

var gapIDReplacer = strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_", ":", "_")

func gapID(resource string) string {
	return strings.ToUpper(gapIDReplacer.Replace(
		upgradev1alpha1.FindingTypeRBACAssessmentGap + "_SNAPSHOT_" + resource))
}

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

package checkers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

// Checker assesses one read-only upgrade-readiness risk category.
type Checker interface {
	Name() string
	Check(ctx context.Context, c client.Client, assessment *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error)
}

// Default returns checkers enabled by the assessment. If no check is explicitly enabled,
// all MVP checkers run.
func Default(assessment *upgradev1alpha1.UpgradeAssessment) []Checker {
	checks := assessment.Spec.Checks
	runAll := !checks.DeprecatedAPIs &&
		!checks.WorkloadAvailability &&
		!checks.PDB &&
		!checks.ReadinessProbes &&
		!checks.AdmissionWebhooks &&
		!checks.PolicyRisks &&
		!checks.Capacity &&
		!checks.Observability

	var selected []Checker
	add := func(enabled bool, checker Checker) {
		if runAll || enabled {
			selected = append(selected, checker)
		}
	}

	add(checks.WorkloadAvailability, WorkloadAvailability{})
	add(checks.ReadinessProbes, ReadinessProbe{})
	add(checks.PDB, PDB{})
	add(checks.AdmissionWebhooks, AdmissionWebhook{})
	add(checks.PolicyRisks, PolicyRisk{})
	add(checks.DeprecatedAPIs, DeprecatedAPI{})
	add(checks.Capacity, Capacity{})
	add(checks.Observability, Observability{})

	return selected
}

func namespaces(ctx context.Context, c client.Client, assessment *upgradev1alpha1.UpgradeAssessment) ([]string, error) {
	include := stringSet(assessment.Spec.Scope.Namespaces.Include)
	exclude := stringSet(assessment.Spec.Scope.Namespaces.Exclude)
	if len(include) > 0 {
		out := make([]string, 0, len(include))
		for ns := range include {
			if _, blocked := exclude[ns]; !blocked {
				out = append(out, ns)
			}
		}
		sort.Strings(out)
		return out, nil
	}

	var list corev1.NamespaceList
	if err := c.List(ctx, &list); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(list.Items))
	for _, ns := range list.Items {
		if _, blocked := exclude[ns.Name]; !blocked {
			out = append(out, ns.Name)
		}
	}
	sort.Strings(out)
	return out, nil
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

func rbacGap(checker string, err error) []upgradev1alpha1.Finding {
	return []upgradev1alpha1.Finding{{
		ID:       findingID(upgradev1alpha1.FindingTypeRBACAssessmentGap, checker),
		Type:     upgradev1alpha1.FindingTypeRBACAssessmentGap,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "RBAC",
		Message:  fmt.Sprintf("%s could not complete because Kubernetes RBAC denied access: %v", checker, err),
		Evidence: []upgradev1alpha1.Evidence{{
			ID:          evidenceID(upgradev1alpha1.FindingTypeRBACAssessmentGap, checker),
			Description: "RBAC denied a read operation required for assessment.",
			Observed: map[string]string{
				"error": err.Error(),
			},
		}},
		Recommendation: "Grant read-only permissions for this resource type, then rerun the assessment.",
	}}
}

func isRBACDenied(err error) bool {
	if err == nil {
		return false
	}
	return apierrors.IsForbidden(err)
}

func resource(apiVersion, kind, namespace, name string) *upgradev1alpha1.ResourceRef {
	return &upgradev1alpha1.ResourceRef{
		APIVersion: apiVersion,
		Kind:       kind,
		Namespace:  namespace,
		Name:       name,
	}
}

func findingID(findingType string, parts ...string) string {
	values := append([]string{findingType}, parts...)
	return strings.ToUpper(sanitizeID(strings.Join(values, "_")))
}

func evidenceID(findingType string, parts ...string) string {
	return findingID(findingType, append([]string{"EVIDENCE"}, parts...)...)
}

func sanitizeID(value string) string {
	replacer := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_", ":", "_")
	return replacer.Replace(value)
}

func gvk(apiVersion, kind string) schema.GroupVersionKind {
	gv, _ := schema.ParseGroupVersion(apiVersion)
	return gv.WithKind(kind)
}

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/checkers"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/export"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/reporting"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

type staticChecker struct {
	findings []upgradev1alpha1.Finding
}

// testScheme registers everything the snapshot collector lists, so a reconcile
// driven by a fake client exercises the same collection path as production.
func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{
		clientgoscheme.AddToScheme,
		apiextensionsv1.AddToScheme,
		upgradev1alpha1.AddToScheme,
	} {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}
	return scheme
}

func TestReconcileBoundsPublishedFindingsAndPlanActions(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)

	assessment := &upgradev1alpha1.UpgradeAssessment{
		ObjectMeta: metav1.ObjectMeta{Name: "large-assessment", Namespace: "default", Generation: 1},
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			TargetVersion: "1.32",
			Mode:          upgradev1alpha1.AssessmentModeReadOnly,
		},
	}

	findings := make([]upgradev1alpha1.Finding, 0, 250)
	for i := 0; i < 250; i++ {
		findings = append(findings, upgradev1alpha1.Finding{
			ID:             fmt.Sprintf("PDB_BLOCKING_%03d", i),
			Type:           upgradev1alpha1.FindingTypePDBBlockingRisk,
			Severity:       upgradev1alpha1.RiskLevelHigh,
			Category:       "PDB",
			Resource:       &upgradev1alpha1.ResourceRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "default", Name: fmt.Sprintf("workload-%03d", i)},
			Recommendation: "Relax the blocking PDB before the upgrade.",
			Evidence:       []upgradev1alpha1.Evidence{{ID: fmt.Sprintf("PDB_BLOCKING_%03d_EVIDENCE", i), Description: "replicas=1 minAvailable=1"}},
		})
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(assessment).
		WithStatusSubresource(&upgradev1alpha1.UpgradeAssessment{}, &upgradev1alpha1.UpgradePlan{}).
		Build()

	reconciler := &UpgradeAssessmentReconciler{
		Client:   k8sClient,
		Scheme:   scheme,
		Checkers: []checkers.Checker{staticChecker{findings: findings}},
	}

	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(assessment)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	var updated upgradev1alpha1.UpgradeAssessment
	if err := k8sClient.Get(ctx, req.NamespacedName, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Summary.TotalFindings != 250 {
		t.Fatalf("expected summary total to preserve all findings, got %d", updated.Status.Summary.TotalFindings)
	}
	if len(updated.Status.Findings) != maxPublishedFindings {
		t.Fatalf("expected %d published findings, got %d", maxPublishedFindings, len(updated.Status.Findings))
	}
	truncated := meta.FindStatusCondition(updated.Status.Conditions, "AssessmentOutputTruncated")
	if truncated == nil || truncated.Status != metav1.ConditionTrue {
		t.Fatalf("expected truncation condition, got %#v", truncated)
	}

	var plans upgradev1alpha1.UpgradePlanList
	if err := k8sClient.List(ctx, &plans, client.InNamespace("default")); err != nil {
		t.Fatal(err)
	}
	if len(plans.Items) != 1 {
		t.Fatalf("expected one plan, got %d", len(plans.Items))
	}
	if len(plans.Items[0].Spec.RequiredActions) != maxPlanActions {
		t.Fatalf("expected %d plan actions, got %d", maxPlanActions, len(plans.Items[0].Spec.RequiredActions))
	}
}

func (s staticChecker) Name() string { return "static" }

func (s staticChecker) Check(*snapshot.ClusterSnapshot, *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	return s.findings
}

func TestReconcileCreatesIdempotentUpgradePlan(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)

	assessment := &upgradev1alpha1.UpgradeAssessment{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-upgrade-assessment", Namespace: "default", Generation: 1},
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			TargetVersion: "1.32",
			Mode:          upgradev1alpha1.AssessmentModeReadOnly,
		},
	}

	finding := upgradev1alpha1.Finding{
		ID:       "PDB_BLOCKING_001",
		Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
		Severity: upgradev1alpha1.RiskLevelCritical,
		Category: "PDB",
		Resource: &upgradev1alpha1.ResourceRef{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "default", Name: "payment-api"},
		Evidence: []upgradev1alpha1.Evidence{{ID: "PDB_BLOCKING_001_EVIDENCE", Description: "replicas=1 minAvailable=1"}},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(assessment).
		WithStatusSubresource(&upgradev1alpha1.UpgradeAssessment{}, &upgradev1alpha1.UpgradePlan{}).
		Build()

	reconciler := &UpgradeAssessmentReconciler{
		Client:   k8sClient,
		Scheme:   scheme,
		Checkers: []checkers.Checker{staticChecker{findings: []upgradev1alpha1.Finding{finding}}},
	}

	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(assessment)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("first reconcile failed: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("second reconcile failed: %v", err)
	}

	var plans upgradev1alpha1.UpgradePlanList
	if err := k8sClient.List(ctx, &plans, client.InNamespace("default")); err != nil {
		t.Fatal(err)
	}
	if len(plans.Items) != 1 {
		t.Fatalf("expected one plan, got %d", len(plans.Items))
	}
	plan := plans.Items[0]
	if plan.Spec.Decision != upgradev1alpha1.DecisionDoNotUpgrade {
		t.Fatalf("expected DoNotUpgrade, got %s", plan.Spec.Decision)
	}
	if len(plan.Spec.RequiredActions) != 1 {
		t.Fatalf("expected one required action, got %d", len(plan.Spec.RequiredActions))
	}

	var updated upgradev1alpha1.UpgradeAssessment
	if err := k8sClient.Get(ctx, req.NamespacedName, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase != upgradev1alpha1.AssessmentPhaseCompleted {
		t.Fatalf("expected completed assessment, got %s", updated.Status.Phase)
	}
	if updated.Status.GeneratedPlanRef == nil || updated.Status.GeneratedPlanRef.Name != "prod-upgrade-assessment-plan" {
		t.Fatalf("expected generated plan ref, got %#v", updated.Status.GeneratedPlanRef)
	}
	if updated.Status.LastAssessedTime == nil {
		t.Fatal("expected the snapshot time to be published as lastAssessedTime")
	}

	artifact := &corev1.ConfigMap{}
	artifactKey := client.ObjectKey{Namespace: "default", Name: "prod-upgrade-assessment-artifact"}
	if err := k8sClient.Get(ctx, artifactKey, artifact); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{reporting.AssessmentMarkdownKey, reporting.PlanMarkdownKey, export.JSONKey} {
		if artifact.Data[key] == "" {
			t.Fatalf("expected artifact key %q in the ConfigMap, got keys %v", key, artifactKeys(artifact))
		}
	}

	var document map[string]any
	if err := json.Unmarshal([]byte(artifact.Data[export.JSONKey]), &document); err != nil {
		t.Fatalf("assessment.json is not valid JSON: %v", err)
	}
	if document["exportVersion"] != export.JSONVersion {
		t.Fatalf("expected exportVersion %q, got %v", export.JSONVersion, document["exportVersion"])
	}
	if document["takenAt"] == nil {
		t.Fatalf("expected takenAt in the export, got %v", document)
	}
}

func artifactKeys(artifact *corev1.ConfigMap) []string {
	keys := make([]string, 0, len(artifact.Data))
	for key := range artifact.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

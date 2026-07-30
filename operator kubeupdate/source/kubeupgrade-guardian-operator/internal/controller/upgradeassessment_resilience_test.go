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
	"errors"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/checkers"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

type panickingChecker struct{}

func (panickingChecker) Name() string { return "exploding" }

func (panickingChecker) Check(*snapshot.ClusterSnapshot, *upgradev1alpha1.UpgradeAssessment) []upgradev1alpha1.Finding {
	panic("nil map write")
}

func TestReconcileKeepsPartialResultsWhenACheckerPanics(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	assessment := newAssessment("resilient", nil)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(assessment).
		WithStatusSubresource(&upgradev1alpha1.UpgradeAssessment{}, &upgradev1alpha1.UpgradePlan{}).
		Build()

	reconciler := &UpgradeAssessmentReconciler{
		Client: k8sClient,
		Scheme: scheme,
		Checkers: []checkers.Checker{
			panickingChecker{},
			staticChecker{findings: []upgradev1alpha1.Finding{{
				ID:       "PDB_BLOCKING_001",
				Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
				Severity: upgradev1alpha1.RiskLevelHigh,
				Category: "PDB",
			}}},
		},
	}

	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(assessment)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("a panicking checker must not fail the reconcile: %v", err)
	}

	updated := &upgradev1alpha1.UpgradeAssessment{}
	if err := k8sClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase != upgradev1alpha1.AssessmentPhaseCompleted {
		t.Fatalf("expected the assessment to complete with partial results, got %s", updated.Status.Phase)
	}

	degraded := meta.FindStatusCondition(updated.Status.Conditions, upgradev1alpha1.ConditionAssessmentDegraded)
	if degraded == nil || degraded.Status != metav1.ConditionTrue {
		t.Fatalf("expected AssessmentDegraded=True, got %#v", degraded)
	}
	if !strings.Contains(degraded.Message, "exploding") {
		t.Fatalf("expected the failing checker to be named, got %q", degraded.Message)
	}

	if !hasFindingOfType(updated.Status.Findings, upgradev1alpha1.FindingTypeAssessmentError) {
		t.Fatalf("expected an ASSESSMENT_ERROR finding, got %#v", updated.Status.Findings)
	}
	if !hasFindingOfType(updated.Status.Findings, upgradev1alpha1.FindingTypePDBBlockingRisk) {
		t.Fatalf("the healthy checker's findings must still be published, got %#v", updated.Status.Findings)
	}
}

func TestReconcileMarksDegradedWhenAResourceTypeCannotBeRead(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	assessment := newAssessment("denied", nil)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(assessment).
		WithStatusSubresource(&upgradev1alpha1.UpgradeAssessment{}, &upgradev1alpha1.UpgradePlan{}).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*appsv1.DeploymentList); ok {
					return apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, "", errors.New("denied"))
				}
				return c.List(ctx, list, opts...)
			},
		}).
		Build()

	reconciler := &UpgradeAssessmentReconciler{Client: k8sClient, Scheme: scheme}

	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(assessment)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatalf("an RBAC gap must not fail the reconcile: %v", err)
	}

	updated := &upgradev1alpha1.UpgradeAssessment{}
	if err := k8sClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase != upgradev1alpha1.AssessmentPhaseCompleted {
		t.Fatalf("expected Completed with partial results, got %s", updated.Status.Phase)
	}
	degraded := meta.FindStatusCondition(updated.Status.Conditions, upgradev1alpha1.ConditionAssessmentDegraded)
	if degraded == nil || degraded.Status != metav1.ConditionTrue {
		t.Fatalf("expected AssessmentDegraded=True, got %#v", degraded)
	}
	if !hasFindingOfType(updated.Status.Findings, upgradev1alpha1.FindingTypeRBACAssessmentGap) {
		t.Fatalf("expected the RBAC gap to be published, got %#v", updated.Status.Findings)
	}
}

func TestReconcileClearsDegradedOnAHealthyRun(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	assessment := newAssessment("healthy", nil)

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(assessment).
		WithStatusSubresource(&upgradev1alpha1.UpgradeAssessment{}, &upgradev1alpha1.UpgradePlan{}).
		Build()

	reconciler := &UpgradeAssessmentReconciler{Client: k8sClient, Scheme: scheme}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(assessment)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}

	updated := &upgradev1alpha1.UpgradeAssessment{}
	if err := k8sClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatal(err)
	}
	degraded := meta.FindStatusCondition(updated.Status.Conditions, upgradev1alpha1.ConditionAssessmentDegraded)
	if degraded == nil || degraded.Status != metav1.ConditionFalse {
		t.Fatalf("expected AssessmentDegraded=False, got %#v", degraded)
	}
}

func TestReconcileRerunsWhenTheRerunAnnotationChanges(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	assessment := newAssessment("rerun", map[string]string{upgradev1alpha1.RerunAnnotation: "build-1"})

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(assessment).
		WithStatusSubresource(&upgradev1alpha1.UpgradeAssessment{}, &upgradev1alpha1.UpgradePlan{}).
		Build()

	reconciler := &UpgradeAssessmentReconciler{Client: k8sClient, Scheme: scheme}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(assessment)}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}

	updated := &upgradev1alpha1.UpgradeAssessment{}
	if err := k8sClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.LastRerunToken != "build-1" {
		t.Fatalf("expected the acted-on token to be remembered, got %q", updated.Status.LastRerunToken)
	}

	first := updated.Status.LastAssessedTime
	if due, _ := assessmentDue(updated, time.Now()); due {
		t.Fatal("an unchanged token must not trigger a new audit")
	}

	updated.Annotations[upgradev1alpha1.RerunAnnotation] = "build-2"
	if due, _ := assessmentDue(updated, time.Now()); !due {
		t.Fatal("a new token must trigger a new audit")
	}
	if err := k8sClient.Update(ctx, updated); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}

	reassessed := &upgradev1alpha1.UpgradeAssessment{}
	if err := k8sClient.Get(ctx, req.NamespacedName, reassessed); err != nil {
		t.Fatal(err)
	}
	if reassessed.Status.LastRerunToken != "build-2" {
		t.Fatalf("expected the new token to be recorded, got %q", reassessed.Status.LastRerunToken)
	}
	if first != nil && reassessed.Status.LastAssessedTime == nil {
		t.Fatal("expected the re-run to refresh lastAssessedTime")
	}
}

func TestAssessmentDueHonoursTheRefreshInterval(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	assessment := newAssessment("refreshing", nil)
	assessment.Spec.RefreshInterval = &metav1.Duration{Duration: 30 * time.Minute}
	assessment.Status.Phase = upgradev1alpha1.AssessmentPhaseCompleted
	meta.SetStatusCondition(&assessment.Status.Conditions, metav1.Condition{
		Type:               upgradev1alpha1.ConditionAssessmentCompleted,
		Status:             metav1.ConditionTrue,
		Reason:             "Completed",
		ObservedGeneration: assessment.Generation,
	})

	lastAssessed := metav1.NewTime(now.Add(-10 * time.Minute))
	assessment.Status.LastAssessedTime = &lastAssessed
	due, requeueAfter := assessmentDue(assessment, now)
	if due {
		t.Fatal("an assessment refreshed 10 minutes ago with a 30 minute interval is not due")
	}
	if requeueAfter != 20*time.Minute {
		t.Fatalf("expected a 20 minute requeue, got %s", requeueAfter)
	}

	lastAssessed = metav1.NewTime(now.Add(-31 * time.Minute))
	assessment.Status.LastAssessedTime = &lastAssessed
	if due, _ := assessmentDue(assessment, now); !due {
		t.Fatal("an assessment older than the refresh interval is due")
	}

	assessment.Status.LastAssessedTime = nil
	if due, _ := assessmentDue(assessment, now); !due {
		t.Fatal("an assessment that never recorded an observation time is due")
	}
}

func TestAssessmentDueStaysQuietWithoutARefreshInterval(t *testing.T) {
	assessment := newAssessment("once", nil)
	assessment.Status.Phase = upgradev1alpha1.AssessmentPhaseCompleted
	meta.SetStatusCondition(&assessment.Status.Conditions, metav1.Condition{
		Type:               upgradev1alpha1.ConditionAssessmentCompleted,
		Status:             metav1.ConditionTrue,
		Reason:             "Completed",
		ObservedGeneration: assessment.Generation,
	})

	due, requeueAfter := assessmentDue(assessment, time.Now())
	if due || requeueAfter != 0 {
		t.Fatalf("expected no re-run and no requeue, got due=%v requeueAfter=%s", due, requeueAfter)
	}
}

func TestRefreshIntervalIsFloored(t *testing.T) {
	assessment := newAssessment("fast", nil)
	assessment.Spec.RefreshInterval = &metav1.Duration{Duration: time.Second}
	if got := refreshInterval(assessment); got != minRefreshInterval {
		t.Fatalf("expected the interval to be raised to %s, got %s", minRefreshInterval, got)
	}

	assessment.Spec.RefreshInterval = &metav1.Duration{Duration: -time.Hour}
	if got := refreshInterval(assessment); got != 0 {
		t.Fatalf("a non-positive interval means on-demand only, got %s", got)
	}

	assessment.Spec.RefreshInterval = nil
	if got := refreshInterval(assessment); got != 0 {
		t.Fatalf("expected 0 without an interval, got %s", got)
	}
}

func TestReconcileRequeuesAtTheRefreshInterval(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme(t)
	assessment := newAssessment("periodic", nil)
	assessment.Spec.RefreshInterval = &metav1.Duration{Duration: time.Hour}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(assessment).
		WithStatusSubresource(&upgradev1alpha1.UpgradeAssessment{}, &upgradev1alpha1.UpgradePlan{}).
		Build()

	reconciler := &UpgradeAssessmentReconciler{Client: k8sClient, Scheme: scheme}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(assessment)}

	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter != time.Hour {
		t.Fatalf("expected a one hour requeue after a completed audit, got %s", result.RequeueAfter)
	}

	result, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter <= 0 || result.RequeueAfter > time.Hour {
		t.Fatalf("expected the next reconcile to wait for the remaining interval, got %s", result.RequeueAfter)
	}
}

func TestReconcileIgnoresADeletedAssessment(t *testing.T) {
	scheme := testScheme(t)
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &UpgradeAssessmentReconciler{Client: k8sClient, Scheme: scheme}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Namespace: "default", Name: "gone"},
	})
	if err != nil {
		t.Fatalf("a deleted assessment must not produce an error: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("expected no requeue for a deleted assessment, got %s", result.RequeueAfter)
	}
}

func newAssessment(name string, annotations map[string]string) *upgradev1alpha1.UpgradeAssessment {
	return &upgradev1alpha1.UpgradeAssessment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "default",
			Generation:  1,
			Annotations: annotations,
		},
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			TargetVersion: "1.32",
			Mode:          upgradev1alpha1.AssessmentModeReadOnly,
		},
	}
}

func hasFindingOfType(findings []upgradev1alpha1.Finding, findingType string) bool {
	for _, finding := range findings {
		if finding.Type == findingType {
			return true
		}
	}
	return false
}

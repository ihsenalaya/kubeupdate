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
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/checkers"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/classifier"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/planner"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/reporting"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/scoring"
)

const (
	maxPublishedFindings = 200
	maxPlanActions       = 200
)

// UpgradeAssessmentReconciler reconciles a UpgradeAssessment object
type UpgradeAssessmentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Checkers []checkers.Checker
}

//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeassessments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeassessments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeassessments/finalizers,verbs=update
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeplans,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeplans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces;nodes;pods;services,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations;mutatingwebhookconfigurations,verbs=get;list;watch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch

// Reconcile executes read-only checks and writes only UpgradeAssessment status
// plus the generated UpgradePlan. It never upgrades, drains, or patches workloads.
func (r *UpgradeAssessmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var assessment upgradev1alpha1.UpgradeAssessment
	if err := r.Get(ctx, req.NamespacedName, &assessment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if completedForCurrentGeneration(&assessment) {
		return ctrl.Result{}, nil
	}

	if err := r.markRunning(ctx, &assessment); err != nil {
		return ctrl.Result{}, err
	}

	rawFindings, err := r.runCheckers(ctx, &assessment)
	if err != nil {
		logger.Error(err, "assessment failed")
		if statusErr := r.markFailed(ctx, &assessment, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	sort.SliceStable(rawFindings, func(i, j int) bool {
		return rawFindings[i].ID < rawFindings[j].ID
	})

	classified := classifier.Classify(rawFindings, assessment.Spec, time.Now())
	effectiveFindings := classified.BlockingFindings()
	score, summary := scoring.Score(effectiveFindings)
	_, rawSummary := scoring.Score(classified.Findings)
	riskLevel := scoring.RiskLevel(score)
	decision := scoring.Decision(score, summary, effectiveFindings)

	planName := assessment.Name + "-plan"
	artifactName := reporting.ArtifactName(&assessment)
	planSpec := planner.BuildSpec(&assessment, decision, riskLevel, score, summary, rawSummary, classified.Summary, boundedFindings(classified.Findings, maxPlanActions))
	if err := r.upsertPlan(ctx, &assessment, planName, planSpec); err != nil {
		if statusErr := r.markFailed(ctx, &assessment, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	if err := r.upsertArtifact(ctx, &assessment, planName, artifactName, riskLevel, score, summary, rawSummary, classified.Summary, classified.Findings, planSpec); err != nil {
		if statusErr := r.markFailed(ctx, &assessment, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.markCompleted(ctx, &assessment, planName, artifactName, riskLevel, score, summary, rawSummary, classified.Summary, classified.Findings)
}

// SetupWithManager sets up the controller with the Manager.
func (r *UpgradeAssessmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&upgradev1alpha1.UpgradeAssessment{}).
		Owns(&upgradev1alpha1.UpgradePlan{}).
		Complete(r)
}

func (r *UpgradeAssessmentReconciler) runCheckers(ctx context.Context, assessment *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, error) {
	selected := r.Checkers
	if len(selected) == 0 {
		selected = checkers.Default(assessment)
	}

	var findings []upgradev1alpha1.Finding
	for _, checker := range selected {
		checkerFindings, err := checker.Check(ctx, r.Client, assessment)
		if err != nil {
			return nil, err
		}
		findings = append(findings, checkerFindings...)
	}
	return findings, nil
}

func (r *UpgradeAssessmentReconciler) markRunning(ctx context.Context, assessment *upgradev1alpha1.UpgradeAssessment) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &upgradev1alpha1.UpgradeAssessment{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(assessment), latest); err != nil {
			return err
		}
		if completedForCurrentGeneration(latest) {
			return nil
		}
		latest.Status.Phase = upgradev1alpha1.AssessmentPhaseRunning
		setCondition(latest, metav1.Condition{
			Type:               upgradev1alpha1.ConditionAssessmentRunning,
			Status:             metav1.ConditionTrue,
			Reason:             "Running",
			Message:            "Assessment is running.",
			ObservedGeneration: latest.Generation,
		})
		return r.Status().Update(ctx, latest)
	})
}

func (r *UpgradeAssessmentReconciler) markFailed(ctx context.Context, assessment *upgradev1alpha1.UpgradeAssessment, err error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &upgradev1alpha1.UpgradeAssessment{}
		if getErr := r.Get(ctx, client.ObjectKeyFromObject(assessment), latest); getErr != nil {
			return getErr
		}
		latest.Status.Phase = upgradev1alpha1.AssessmentPhaseFailed
		setCondition(latest, metav1.Condition{
			Type:               upgradev1alpha1.ConditionAssessmentFailed,
			Status:             metav1.ConditionTrue,
			Reason:             "Error",
			Message:            err.Error(),
			ObservedGeneration: latest.Generation,
		})
		return r.Status().Update(ctx, latest)
	})
}

func (r *UpgradeAssessmentReconciler) markCompleted(
	ctx context.Context,
	assessment *upgradev1alpha1.UpgradeAssessment,
	planName string,
	artifactName string,
	riskLevel upgradev1alpha1.RiskLevel,
	score int,
	summary upgradev1alpha1.FindingSummary,
	rawSummary upgradev1alpha1.FindingSummary,
	classificationSummary upgradev1alpha1.ClassificationSummary,
	findings []upgradev1alpha1.Finding,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &upgradev1alpha1.UpgradeAssessment{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(assessment), latest); err != nil {
			return err
		}
		latest.Status.Phase = upgradev1alpha1.AssessmentPhaseCompleted
		latest.Status.RiskLevel = riskLevel
		latest.Status.Score = score
		latest.Status.Summary = summary
		latest.Status.RawSummary = rawSummary
		latest.Status.ClassificationSummary = classificationSummary
		latest.Status.Findings = boundedFindings(findings, maxPublishedFindings)
		latest.Status.GeneratedPlanRef = &upgradev1alpha1.PlanReference{Name: planName}
		latest.Status.ArtifactRef = &upgradev1alpha1.ArtifactReference{
			Kind:      "ConfigMap",
			Name:      artifactName,
			Namespace: latest.Namespace,
		}
		setCondition(latest, metav1.Condition{
			Type:               upgradev1alpha1.ConditionAssessmentCompleted,
			Status:             metav1.ConditionTrue,
			Reason:             "Completed",
			Message:            "Assessment completed successfully.",
			ObservedGeneration: latest.Generation,
		})
		setCondition(latest, metav1.Condition{
			Type:               upgradev1alpha1.ConditionAssessmentRunning,
			Status:             metav1.ConditionFalse,
			Reason:             "Completed",
			Message:            "Assessment is not running.",
			ObservedGeneration: latest.Generation,
		})
		setTruncationCondition(latest, findings)
		return r.Status().Update(ctx, latest)
	})
}

func (r *UpgradeAssessmentReconciler) upsertArtifact(
	ctx context.Context,
	assessment *upgradev1alpha1.UpgradeAssessment,
	planName string,
	artifactName string,
	riskLevel upgradev1alpha1.RiskLevel,
	score int,
	summary upgradev1alpha1.FindingSummary,
	rawSummary upgradev1alpha1.FindingSummary,
	classificationSummary upgradev1alpha1.ClassificationSummary,
	findings []upgradev1alpha1.Finding,
	planSpec upgradev1alpha1.UpgradePlanSpec,
) error {
	renderedAssessment := assessment.DeepCopy()
	renderedAssessment.Status.Phase = upgradev1alpha1.AssessmentPhaseCompleted
	renderedAssessment.Status.RiskLevel = riskLevel
	renderedAssessment.Status.Score = score
	renderedAssessment.Status.Summary = summary
	renderedAssessment.Status.RawSummary = rawSummary
	renderedAssessment.Status.ClassificationSummary = classificationSummary
	renderedAssessment.Status.Findings = boundedFindings(findings, maxPublishedFindings)
	renderedAssessment.Status.GeneratedPlanRef = &upgradev1alpha1.PlanReference{Name: planName}
	renderedAssessment.Status.ArtifactRef = &upgradev1alpha1.ArtifactReference{
		Kind:      "ConfigMap",
		Name:      artifactName,
		Namespace: assessment.Namespace,
	}

	renderedPlan := &upgradev1alpha1.UpgradePlan{}
	renderedPlan.Namespace = assessment.Namespace
	renderedPlan.Name = planName
	renderedPlan.Spec = planSpec

	configMap := &corev1.ConfigMap{}
	configMap.Namespace = assessment.Namespace
	configMap.Name = artifactName

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		if err := controllerutil.SetControllerReference(assessment, configMap, r.Scheme); err != nil {
			return err
		}
		if configMap.Labels == nil {
			configMap.Labels = map[string]string{}
		}
		configMap.Labels["app.kubernetes.io/name"] = "kubeupgrade-guardian-operator"
		configMap.Labels["app.kubernetes.io/component"] = "assessment-artifact"
		configMap.Labels["upgrade.guardian.io/assessment"] = assessment.Name
		configMap.Data = map[string]string{
			reporting.AssessmentMarkdownKey: reporting.AssessmentMarkdown(renderedAssessment),
			reporting.PlanMarkdownKey:       reporting.PlanMarkdown(renderedPlan),
		}
		return nil
	})
	return err
}

func (r *UpgradeAssessmentReconciler) upsertPlan(ctx context.Context, assessment *upgradev1alpha1.UpgradeAssessment, planName string, spec upgradev1alpha1.UpgradePlanSpec) error {
	plan := &upgradev1alpha1.UpgradePlan{}
	plan.Namespace = assessment.Namespace
	plan.Name = planName

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, plan, func() error {
		if err := controllerutil.SetControllerReference(assessment, plan, r.Scheme); err != nil {
			return err
		}
		plan.Spec = spec
		return nil
	})
	if err != nil {
		return err
	}

	latest := &upgradev1alpha1.UpgradePlan{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(plan), latest); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	now := metav1.Now()
	latest.Status.ObservedGeneration = latest.Generation
	latest.Status.GeneratedAt = &now
	setPlanCondition(latest, metav1.Condition{
		Type:               "PlanGenerated",
		Status:             metav1.ConditionTrue,
		Reason:             "Generated",
		Message:            "UpgradePlan generated from latest assessment.",
		ObservedGeneration: latest.Generation,
	})
	return r.Status().Update(ctx, latest)
}

func setCondition(assessment *upgradev1alpha1.UpgradeAssessment, condition metav1.Condition) {
	meta.SetStatusCondition(&assessment.Status.Conditions, condition)
}

func setTruncationCondition(assessment *upgradev1alpha1.UpgradeAssessment, findings []upgradev1alpha1.Finding) {
	if len(findings) <= maxPublishedFindings && nonInfoFindingCount(findings) <= maxPlanActions {
		setCondition(assessment, metav1.Condition{
			Type:               "AssessmentOutputTruncated",
			Status:             metav1.ConditionFalse,
			Reason:             "WithinLimit",
			Message:            "All findings and plan actions are published.",
			ObservedGeneration: assessment.Generation,
		})
		return
	}

	setCondition(assessment, metav1.Condition{
		Type:               "AssessmentOutputTruncated",
		Status:             metav1.ConditionTrue,
		Reason:             "BoundedStatus",
		Message:            fmt.Sprintf("Published %d/%d findings in status and at most %d plan actions to keep Kubernetes objects below API-server size limits.", min(len(findings), maxPublishedFindings), len(findings), maxPlanActions),
		ObservedGeneration: assessment.Generation,
	})
}

func boundedFindings(findings []upgradev1alpha1.Finding, limit int) []upgradev1alpha1.Finding {
	if len(findings) <= limit {
		return append([]upgradev1alpha1.Finding{}, findings...)
	}
	bounded := append([]upgradev1alpha1.Finding{}, findings...)
	sort.SliceStable(bounded, func(i, j int) bool {
		left := severityRank(bounded[i].Severity)
		right := severityRank(bounded[j].Severity)
		if left != right {
			return left < right
		}
		return bounded[i].ID < bounded[j].ID
	})
	return append([]upgradev1alpha1.Finding{}, bounded[:limit]...)
}

func severityRank(level upgradev1alpha1.RiskLevel) int {
	switch level {
	case upgradev1alpha1.RiskLevelCritical:
		return 0
	case upgradev1alpha1.RiskLevelHigh:
		return 1
	case upgradev1alpha1.RiskLevelMedium:
		return 2
	case upgradev1alpha1.RiskLevelLow:
		return 3
	case upgradev1alpha1.RiskLevelInfo:
		return 4
	default:
		return 5
	}
}

func nonInfoFindingCount(findings []upgradev1alpha1.Finding) int {
	count := 0
	for _, finding := range findings {
		if finding.Severity != upgradev1alpha1.RiskLevelInfo {
			count++
		}
	}
	return count
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func completedForCurrentGeneration(assessment *upgradev1alpha1.UpgradeAssessment) bool {
	if assessment.Status.Phase != upgradev1alpha1.AssessmentPhaseCompleted {
		return false
	}
	condition := meta.FindStatusCondition(assessment.Status.Conditions, upgradev1alpha1.ConditionAssessmentCompleted)
	return condition != nil &&
		condition.Status == metav1.ConditionTrue &&
		condition.ObservedGeneration == assessment.Generation
}

func setPlanCondition(plan *upgradev1alpha1.UpgradePlan, condition metav1.Condition) {
	meta.SetStatusCondition(&plan.Status.Conditions, condition)
}

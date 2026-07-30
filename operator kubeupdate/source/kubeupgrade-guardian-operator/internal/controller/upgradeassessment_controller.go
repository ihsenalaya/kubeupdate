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
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/checkers"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/classifier"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/export"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/metrics"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/planner"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/reporting"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/scoring"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

const (
	maxPublishedFindings = 200
	maxPlanActions       = 200

	// minRefreshInterval keeps a mistyped spec.refreshInterval from turning the
	// operator into a cluster-wide polling loop.
	minRefreshInterval = time.Minute
)

// UpgradeAssessmentReconciler reconciles a UpgradeAssessment object
type UpgradeAssessmentReconciler struct {
	client.Client
	// Reader reads the assessed cluster state. It is wired to the manager's API
	// reader so snapshot collection bypasses the cache and starts no informer.
	// When nil, the reconciler falls back to Client.
	Reader   client.Reader
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Checkers []checkers.Checker
}

// assessmentResult is everything one audit produced, computed once and reused by
// the plan, the artifact, the status and the metrics.
type assessmentResult struct {
	TakenAt   metav1.Time
	Findings  []upgradev1alpha1.Finding
	Published []upgradev1alpha1.Finding
	PlanInput []upgradev1alpha1.Finding

	Summary               upgradev1alpha1.FindingSummary
	RawSummary            upgradev1alpha1.FindingSummary
	ClassificationSummary upgradev1alpha1.ClassificationSummary

	Score     int
	RiskLevel upgradev1alpha1.RiskLevel
	Decision  upgradev1alpha1.Decision

	// Degraded is true when part of the cluster could not be read or a checker
	// failed, so the findings below are partial.
	Degraded       bool
	DegradedReason string
}

//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeassessments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeassessments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeassessments/finalizers,verbs=update
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeplans,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=upgrade.guardian.io,resources=upgradeplans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces;nodes;pods;services,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations;mutatingwebhookconfigurations,verbs=get;list;watch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch

// Reconcile executes read-only checks and writes only UpgradeAssessment status
// plus the generated UpgradePlan. It never upgrades, drains, or patches workloads.
func (r *UpgradeAssessmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var assessment upgradev1alpha1.UpgradeAssessment
	if err := r.Get(ctx, req.NamespacedName, &assessment); err != nil {
		if apierrors.IsNotFound(err) {
			// The object is gone; its series would otherwise stay exported forever.
			metrics.ForgetAssessment(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if due, requeueAfter := assessmentDue(&assessment, time.Now()); !due {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	if err := r.markRunning(ctx, &assessment); err != nil {
		return ctrl.Result{}, err
	}

	snap, err := snapshot.Collect(ctx, r.reader(), assessment.Spec.Scope)
	if err != nil {
		logger.Error(err, "assessment failed")
		r.event(&assessment, corev1.EventTypeWarning, "AssessmentFailed", fmt.Sprintf("Cluster snapshot failed: %v", err))
		if statusErr := r.markFailed(ctx, &assessment, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	result := r.assess(snap, &assessment)

	planName := assessment.Name + "-plan"
	artifactName := reporting.ArtifactName(&assessment)
	planSpec := planner.BuildSpec(&assessment, result.Decision, result.RiskLevel, result.Score, result.Summary, result.RawSummary, result.ClassificationSummary, result.PlanInput)
	if err := r.upsertPlan(ctx, &assessment, planName, planSpec); err != nil {
		r.event(&assessment, corev1.EventTypeWarning, "AssessmentFailed", fmt.Sprintf("UpgradePlan could not be written: %v", err))
		if statusErr := r.markFailed(ctx, &assessment, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	if err := r.upsertArtifact(ctx, &assessment, planName, artifactName, result, planSpec); err != nil {
		r.event(&assessment, corev1.EventTypeWarning, "AssessmentFailed", fmt.Sprintf("Assessment artifact could not be written: %v", err))
		if statusErr := r.markFailed(ctx, &assessment, err); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	if err := r.markCompleted(ctx, &assessment, planName, artifactName, result); err != nil {
		return ctrl.Result{}, err
	}

	metrics.RecordAssessment(assessment.Namespace, assessment.Name, result.Score, result.Decision, result.RiskLevel, result.Summary)
	r.event(&assessment, corev1.EventTypeNormal, "AssessmentCompleted",
		fmt.Sprintf("Assessment completed: decision %s, score %d, risk %s.", result.Decision, result.Score, result.RiskLevel))
	if result.Degraded {
		r.event(&assessment, corev1.EventTypeWarning, "AssessmentDegraded", result.DegradedReason)
	}

	return ctrl.Result{RequeueAfter: refreshInterval(&assessment)}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UpgradeAssessmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("upgradeassessment-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&upgradev1alpha1.UpgradeAssessment{}).
		Owns(&upgradev1alpha1.UpgradePlan{}).
		Complete(r)
}

// assess runs the checkers and derives everything the rest of the reconcile needs,
// so the classified findings are sorted and bounded exactly once.
func (r *UpgradeAssessmentReconciler) assess(snap *snapshot.ClusterSnapshot, assessment *upgradev1alpha1.UpgradeAssessment) assessmentResult {
	rawFindings, failedCheckers := r.runCheckers(snap, assessment)
	sort.SliceStable(rawFindings, func(i, j int) bool {
		return rawFindings[i].ID < rawFindings[j].ID
	})

	classified := classifier.Classify(rawFindings, assessment.Spec, time.Now())
	blocking := classified.BlockingFindings()
	score, summary := scoring.Score(blocking)
	_, rawSummary := scoring.Score(classified.Findings)

	result := assessmentResult{
		TakenAt:               metav1.NewTime(snap.TakenAt),
		Findings:              classified.Findings,
		Published:             boundedFindings(classified.Findings, maxPublishedFindings),
		PlanInput:             boundedFindings(classified.Findings, maxPlanActions),
		Summary:               summary,
		RawSummary:            rawSummary,
		ClassificationSummary: classified.Summary,
		Score:                 score,
		RiskLevel:             scoring.RiskLevel(score),
		Decision:              scoring.Decision(score, summary, blocking),
	}

	switch {
	case len(failedCheckers) > 0 && len(snap.Gaps) > 0:
		result.Degraded = true
		result.DegradedReason = fmt.Sprintf("%d checker(s) failed and %d resource type(s) could not be read; the published findings are partial.",
			len(failedCheckers), len(snap.Gaps))
	case len(failedCheckers) > 0:
		result.Degraded = true
		result.DegradedReason = fmt.Sprintf("Checker(s) %s failed; the published findings are partial.", strings.Join(failedCheckers, ", "))
	case len(snap.Gaps) > 0:
		result.Degraded = true
		result.DegradedReason = fmt.Sprintf("%d resource type(s) could not be read; the published findings are partial.", len(snap.Gaps))
	}

	return result
}

// assessmentDue answers whether the current status already covers this generation,
// this re-run token and this refresh interval. When it does not have to run again
// yet, the returned delay is when the next refresh falls due.
func assessmentDue(assessment *upgradev1alpha1.UpgradeAssessment, now time.Time) (bool, time.Duration) {
	if !completedForCurrentGeneration(assessment) {
		return true, 0
	}
	if assessment.Annotations[upgradev1alpha1.RerunAnnotation] != assessment.Status.LastRerunToken {
		return true, 0
	}

	interval := refreshInterval(assessment)
	if interval == 0 {
		return false, 0
	}
	if assessment.Status.LastAssessedTime == nil {
		return true, 0
	}
	if remaining := assessment.Status.LastAssessedTime.Add(interval).Sub(now); remaining > 0 {
		return false, remaining
	}
	return true, 0
}

// refreshInterval returns the effective re-run period, 0 when the assessment runs
// on demand only.
func refreshInterval(assessment *upgradev1alpha1.UpgradeAssessment) time.Duration {
	if assessment.Spec.RefreshInterval == nil || assessment.Spec.RefreshInterval.Duration <= 0 {
		return 0
	}
	if assessment.Spec.RefreshInterval.Duration < minRefreshInterval {
		return minRefreshInterval
	}
	return assessment.Spec.RefreshInterval.Duration
}

func (r *UpgradeAssessmentReconciler) event(assessment *upgradev1alpha1.UpgradeAssessment, eventType, reason, message string) {
	if r.Recorder == nil {
		return
	}
	r.Recorder.Event(assessment, eventType, reason, message)
}

// reader returns the direct cluster reader, falling back to the cached client so
// that tests can drive the reconciler with a single fake client.
func (r *UpgradeAssessmentReconciler) reader() client.Reader {
	if r.Reader != nil {
		return r.Reader
	}
	return r.Client
}

// runCheckers evaluates every enabled checker against the same snapshot. Checkers
// are pure functions with no shared state, so they run concurrently; the caller
// re-sorts the merged result by ID to keep publication deterministic. It returns
// the names of the checkers that did not complete.
func (r *UpgradeAssessmentReconciler) runCheckers(snap *snapshot.ClusterSnapshot, assessment *upgradev1alpha1.UpgradeAssessment) ([]upgradev1alpha1.Finding, []string) {
	selected := r.Checkers
	if len(selected) == 0 {
		selected = checkers.Default(assessment)
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		findings []upgradev1alpha1.Finding
		failed   []string
	)
	for _, checker := range selected {
		checker := checker
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkerFindings, ok := runChecker(checker, snap, assessment)
			mu.Lock()
			defer mu.Unlock()
			findings = append(findings, checkerFindings...)
			if !ok {
				failed = append(failed, checker.Name())
			}
		}()
	}
	wg.Wait()
	sort.Strings(failed)

	// Collection gaps are reported once for the whole assessment rather than once
	// per checker that would have consumed the denied resource type.
	return append(findings, snap.Gaps...), failed
}

// runChecker isolates one checker: a panic becomes an ASSESSMENT_ERROR finding
// instead of taking the whole audit down, so the operator still publishes the
// risks it did manage to observe.
func runChecker(checker checkers.Checker, snap *snapshot.ClusterSnapshot, assessment *upgradev1alpha1.UpgradeAssessment) (findings []upgradev1alpha1.Finding, ok bool) {
	start := time.Now()
	defer func() {
		metrics.ObserveChecker(checker.Name(), time.Since(start))
		if recovered := recover(); recovered != nil {
			findings = []upgradev1alpha1.Finding{
				checkers.AssessmentErrorFinding(checker.Name(), fmt.Errorf("checker panicked: %v", recovered)),
			}
			ok = false
		}
	}()

	return checker.Check(snap, assessment), true
}

func (r *UpgradeAssessmentReconciler) markRunning(ctx context.Context, assessment *upgradev1alpha1.UpgradeAssessment) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &upgradev1alpha1.UpgradeAssessment{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(assessment), latest); err != nil {
			return err
		}
		if alreadyRunning(latest) {
			// Nothing to say: skipping the write avoids a status update - and the
			// reconcile it triggers - on every re-run.
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
	result assessmentResult,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &upgradev1alpha1.UpgradeAssessment{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(assessment), latest); err != nil {
			return err
		}
		applyResultToStatus(latest, planName, artifactName, result)
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
		setDegradedCondition(latest, result)
		setTruncationCondition(latest, result.Findings)
		return r.Status().Update(ctx, latest)
	})
}

// applyResultToStatus writes one audit onto a status. It is shared by the status
// update and by the assessment rendered into the artifact, so the published
// ConfigMap can never disagree with the object it describes.
func applyResultToStatus(assessment *upgradev1alpha1.UpgradeAssessment, planName, artifactName string, result assessmentResult) {
	assessment.Status.Phase = upgradev1alpha1.AssessmentPhaseCompleted
	assessment.Status.RiskLevel = result.RiskLevel
	assessment.Status.Score = result.Score
	assessment.Status.Summary = result.Summary
	assessment.Status.RawSummary = result.RawSummary
	assessment.Status.ClassificationSummary = result.ClassificationSummary
	assessment.Status.Findings = result.Published
	assessment.Status.LastAssessedTime = result.TakenAt.DeepCopy()
	assessment.Status.LastRerunToken = assessment.Annotations[upgradev1alpha1.RerunAnnotation]
	assessment.Status.GeneratedPlanRef = &upgradev1alpha1.PlanReference{Name: planName}
	assessment.Status.ArtifactRef = &upgradev1alpha1.ArtifactReference{
		Kind:      "ConfigMap",
		Name:      artifactName,
		Namespace: assessment.Namespace,
	}
}

func (r *UpgradeAssessmentReconciler) upsertArtifact(
	ctx context.Context,
	assessment *upgradev1alpha1.UpgradeAssessment,
	planName string,
	artifactName string,
	result assessmentResult,
	planSpec upgradev1alpha1.UpgradePlanSpec,
) error {
	renderedAssessment := assessment.DeepCopy()
	applyResultToStatus(renderedAssessment, planName, artifactName, result)

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

		data := make(map[string]string, len(export.Default()))
		for _, renderer := range export.Default() {
			rendered, renderErr := renderer.Render(renderedAssessment, renderedPlan)
			if renderErr != nil {
				return renderErr
			}
			data[renderer.Key()] = string(rendered)
		}
		configMap.Data = data
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

// setDegradedCondition records whether the published findings are complete. The
// phase stays Completed either way: partial results are worth more than no result.
func setDegradedCondition(assessment *upgradev1alpha1.UpgradeAssessment, result assessmentResult) {
	if !result.Degraded {
		setCondition(assessment, metav1.Condition{
			Type:               upgradev1alpha1.ConditionAssessmentDegraded,
			Status:             metav1.ConditionFalse,
			Reason:             "Complete",
			Message:            "Every checker completed against a fully readable cluster.",
			ObservedGeneration: assessment.Generation,
		})
		return
	}

	setCondition(assessment, metav1.Condition{
		Type:               upgradev1alpha1.ConditionAssessmentDegraded,
		Status:             metav1.ConditionTrue,
		Reason:             "PartialResults",
		Message:            result.DegradedReason,
		ObservedGeneration: assessment.Generation,
	})
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

func alreadyRunning(assessment *upgradev1alpha1.UpgradeAssessment) bool {
	if assessment.Status.Phase != upgradev1alpha1.AssessmentPhaseRunning {
		return false
	}
	condition := meta.FindStatusCondition(assessment.Status.Conditions, upgradev1alpha1.ConditionAssessmentRunning)
	return condition != nil &&
		condition.Status == metav1.ConditionTrue &&
		condition.ObservedGeneration == assessment.Generation
}

func setPlanCondition(plan *upgradev1alpha1.UpgradePlan, condition metav1.Condition) {
	meta.SetStatusCondition(&plan.Status.Conditions, condition)
}

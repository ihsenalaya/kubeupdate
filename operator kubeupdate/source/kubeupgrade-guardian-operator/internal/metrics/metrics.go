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

// Package metrics publishes the assessment outcome on the manager's existing
// /metrics endpoint, so upgrade readiness can be alerted on and trended without
// polling the API server for UpgradeAssessment objects.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

var (
	assessmentScore = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "guardian_assessment_score",
		Help: "Weighted risk score of the last completed assessment. Higher is riskier.",
	}, []string{"namespace", "name"})

	assessmentFindings = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "guardian_assessment_findings",
		Help: "Number of published findings of the last completed assessment, by severity.",
	}, []string{"namespace", "name", "severity"})

	checkerDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "guardian_checker_duration_seconds",
		Help:    "Time spent evaluating one checker against the cluster snapshot.",
		Buckets: prometheus.DefBuckets,
	}, []string{"checker"})

	assessmentInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "guardian_assessment_info",
		Help: "Always 1. Carries the decision and risk level of the last completed assessment as labels.",
	}, []string{"namespace", "name", "decision", "risk_level"})
)

func init() {
	ctrlmetrics.Registry.MustRegister(assessmentScore, assessmentFindings, checkerDuration, assessmentInfo)
}

// ObserveChecker records how long a checker took.
func ObserveChecker(checker string, elapsed time.Duration) {
	checkerDuration.WithLabelValues(checker).Observe(elapsed.Seconds())
}

// RecordAssessment publishes the outcome of a completed assessment.
func RecordAssessment(
	namespace, name string,
	score int,
	decision upgradev1alpha1.Decision,
	riskLevel upgradev1alpha1.RiskLevel,
	summary upgradev1alpha1.FindingSummary,
) {
	assessmentScore.WithLabelValues(namespace, name).Set(float64(score))

	for severity, count := range map[upgradev1alpha1.RiskLevel]int{
		upgradev1alpha1.RiskLevelCritical: summary.Critical,
		upgradev1alpha1.RiskLevelHigh:     summary.High,
		upgradev1alpha1.RiskLevelMedium:   summary.Medium,
		upgradev1alpha1.RiskLevelLow:      summary.Low,
		upgradev1alpha1.RiskLevelInfo:     summary.Info,
	} {
		assessmentFindings.WithLabelValues(namespace, name, string(severity)).Set(float64(count))
	}

	// The decision and risk level are labels, so the previous combination has to
	// go: otherwise a cluster that moves from DoNotUpgrade to Proceed would export
	// both series at 1 forever.
	assessmentInfo.DeletePartialMatch(objectLabels(namespace, name))
	assessmentInfo.WithLabelValues(namespace, name, string(decision), string(riskLevel)).Set(1)
}

// ForgetAssessment drops every series of an assessment that no longer exists.
func ForgetAssessment(namespace, name string) {
	labels := objectLabels(namespace, name)
	assessmentScore.DeletePartialMatch(labels)
	assessmentFindings.DeletePartialMatch(labels)
	assessmentInfo.DeletePartialMatch(labels)
}

func objectLabels(namespace, name string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "name": name}
}

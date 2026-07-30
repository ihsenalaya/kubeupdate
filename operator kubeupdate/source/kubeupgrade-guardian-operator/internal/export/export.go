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

// Package export turns a completed assessment and its plan into the artifacts
// published in the assessment ConfigMap. Each output format is a Renderer, so
// adding a format never touches the controller.
package export

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/reporting"
)

// Renderer produces one artifact of the assessment ConfigMap.
type Renderer interface {
	// Key is the ConfigMap data key the artifact is published under.
	Key() string
	Render(assessment *upgradev1alpha1.UpgradeAssessment, plan *upgradev1alpha1.UpgradePlan) ([]byte, error)
}

// JSONKey is the ConfigMap key holding the machine-readable export.
const JSONKey = "assessment.json"

// JSONVersion versions the envelope below. It changes only when the envelope
// changes shape, never when a value changes.
const JSONVersion = "v1"

// Default returns the renderers published for every assessment.
func Default() []Renderer {
	return []Renderer{
		AssessmentMarkdown{},
		PlanMarkdown{},
		JSON{},
	}
}

// AssessmentMarkdown publishes the human-readable assessment summary.
type AssessmentMarkdown struct{}

func (AssessmentMarkdown) Key() string { return reporting.AssessmentMarkdownKey }

func (AssessmentMarkdown) Render(assessment *upgradev1alpha1.UpgradeAssessment, _ *upgradev1alpha1.UpgradePlan) ([]byte, error) {
	return []byte(reporting.AssessmentMarkdown(assessment)), nil
}

// PlanMarkdown publishes the operator-grade upgrade plan.
type PlanMarkdown struct{}

func (PlanMarkdown) Key() string { return reporting.PlanMarkdownKey }

func (PlanMarkdown) Render(_ *upgradev1alpha1.UpgradeAssessment, plan *upgradev1alpha1.UpgradePlan) ([]byte, error) {
	return []byte(reporting.PlanMarkdown(plan)), nil
}

// ObjectRef identifies one of the objects the export was built from.
type ObjectRef struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// Document is the stable envelope of the machine-readable export. Leaf shapes
// (summaries, findings, actions, upgrade path) are the v1alpha1 API types, whose
// JSON representation is already the CRD contract; exportVersion tracks the
// envelope itself.
type Document struct {
	ExportVersion string       `json:"exportVersion"`
	TakenAt       *metav1.Time `json:"takenAt,omitempty"`

	Assessment ObjectRef  `json:"assessment"`
	Plan       *ObjectRef `json:"plan,omitempty"`

	SourceVersion string                            `json:"sourceVersion"`
	TargetVersion string                            `json:"targetVersion"`
	Profile       upgradev1alpha1.AssessmentProfile `json:"profile"`

	Decision  upgradev1alpha1.Decision  `json:"decision"`
	RiskLevel upgradev1alpha1.RiskLevel `json:"riskLevel"`
	Score     int                       `json:"score"`

	Summary               upgradev1alpha1.FindingSummary        `json:"summary"`
	RawSummary            upgradev1alpha1.FindingSummary        `json:"rawSummary"`
	ClassificationSummary upgradev1alpha1.ClassificationSummary `json:"classificationSummary"`

	// Findings are the findings published in the assessment status, i.e. the same
	// bounded set the Markdown artifacts render.
	Findings         []upgradev1alpha1.Finding         `json:"findings"`
	RequiredActions  []upgradev1alpha1.RequiredAction  `json:"requiredActions"`
	UpgradePath      []upgradev1alpha1.UpgradePathStep `json:"upgradePath"`
	RecommendedOrder []string                          `json:"recommendedOrder"`
}

// JSON publishes the machine-readable export consumed by CI and dashboards.
type JSON struct{}

func (JSON) Key() string { return JSONKey }

func (JSON) Render(assessment *upgradev1alpha1.UpgradeAssessment, plan *upgradev1alpha1.UpgradePlan) ([]byte, error) {
	document := Document{
		ExportVersion: JSONVersion,
		TakenAt:       assessment.Status.LastAssessedTime,
		Assessment: ObjectRef{
			Namespace: assessment.Namespace,
			Name:      assessment.Name,
		},
		SourceVersion:         assessment.Spec.SourceVersion,
		TargetVersion:         assessment.Spec.TargetVersion,
		Profile:               assessment.Spec.Profile,
		RiskLevel:             assessment.Status.RiskLevel,
		Score:                 assessment.Status.Score,
		Summary:               assessment.Status.Summary,
		RawSummary:            assessment.Status.RawSummary,
		ClassificationSummary: assessment.Status.ClassificationSummary,
		Findings:              nonNilFindings(assessment.Status.Findings),
		RequiredActions:       []upgradev1alpha1.RequiredAction{},
		UpgradePath:           []upgradev1alpha1.UpgradePathStep{},
		RecommendedOrder:      []string{},
	}

	if plan != nil {
		document.Plan = &ObjectRef{Namespace: plan.Namespace, Name: plan.Name}
		document.Decision = plan.Spec.Decision
		if plan.Spec.RequiredActions != nil {
			document.RequiredActions = plan.Spec.RequiredActions
		}
		if plan.Spec.UpgradePath != nil {
			document.UpgradePath = plan.Spec.UpgradePath
		}
		if plan.Spec.RecommendedOrder != nil {
			document.RecommendedOrder = plan.Spec.RecommendedOrder
		}
	}

	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func nonNilFindings(findings []upgradev1alpha1.Finding) []upgradev1alpha1.Finding {
	if findings == nil {
		return []upgradev1alpha1.Finding{}
	}
	return findings
}

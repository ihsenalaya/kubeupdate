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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceScope filters namespaced resources included in an assessment.
type NamespaceScope struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// AssessmentScope defines the cluster slice to assess.
type AssessmentScope struct {
	Namespaces NamespaceScope `json:"namespaces,omitempty"`
}

// AssessmentChecks enables or disables individual assessment checkers.
type AssessmentChecks struct {
	DeprecatedAPIs       bool `json:"deprecatedApis,omitempty"`
	WorkloadAvailability bool `json:"workloadAvailability,omitempty"`
	PDB                  bool `json:"pdb,omitempty"`
	ReadinessProbes      bool `json:"readinessProbes,omitempty"`
	AdmissionWebhooks    bool `json:"admissionWebhooks,omitempty"`
	PolicyRisks          bool `json:"policyRisks,omitempty"`
	Capacity             bool `json:"capacity,omitempty"`
	Observability        bool `json:"observability,omitempty"`
}

// UpgradeAssessmentSpec defines the desired state of UpgradeAssessment
type UpgradeAssessmentSpec struct {
	// TargetVersion is the Kubernetes minor version being assessed, for example "1.32".
	// +kubebuilder:validation:Pattern=`^1\.[0-9]+$`
	TargetVersion string `json:"targetVersion"`

	// Mode must be ReadOnly. The controller never performs upgrades, drains, or workload patches.
	// +kubebuilder:default=ReadOnly
	Mode AssessmentMode `json:"mode,omitempty"`

	Scope  AssessmentScope  `json:"scope,omitempty"`
	Checks AssessmentChecks `json:"checks,omitempty"`
}

// UpgradeAssessmentStatus defines the observed state of UpgradeAssessment
type UpgradeAssessmentStatus struct {
	Phase            AssessmentPhase `json:"phase,omitempty"`
	RiskLevel        RiskLevel       `json:"riskLevel,omitempty"`
	Score            int             `json:"score,omitempty"`
	Summary          FindingSummary  `json:"summary,omitempty"`
	Findings         []Finding       `json:"findings,omitempty"`
	GeneratedPlanRef *PlanReference  `json:"generatedPlanRef,omitempty"`

	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// UpgradeAssessment is the Schema for the upgradeassessments API
type UpgradeAssessment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UpgradeAssessmentSpec   `json:"spec,omitempty"`
	Status UpgradeAssessmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UpgradeAssessmentList contains a list of UpgradeAssessment
type UpgradeAssessmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UpgradeAssessment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UpgradeAssessment{}, &UpgradeAssessmentList{})
}

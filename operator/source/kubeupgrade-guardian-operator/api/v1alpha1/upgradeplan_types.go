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

// UpgradePlanSpec defines the desired state of UpgradePlan
type UpgradePlanSpec struct {
	AssessmentRef    AssessmentReference `json:"assessmentRef"`
	Decision         Decision            `json:"decision"`
	RiskLevel        RiskLevel           `json:"riskLevel"`
	Score            int                 `json:"score,omitempty"`
	Summary          FindingSummary      `json:"summary,omitempty"`
	RequiredActions  []RequiredAction    `json:"requiredActions,omitempty"`
	RecommendedOrder []string            `json:"recommendedOrder,omitempty"`
}

// UpgradePlanStatus defines the observed state of UpgradePlan
type UpgradePlanStatus struct {
	ObservedGeneration int64        `json:"observedGeneration,omitempty"`
	GeneratedAt        *metav1.Time `json:"generatedAt,omitempty"`

	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// UpgradePlan is the Schema for the upgradeplans API
type UpgradePlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UpgradePlanSpec   `json:"spec,omitempty"`
	Status UpgradePlanStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UpgradePlanList contains a list of UpgradePlan
type UpgradePlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UpgradePlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UpgradePlan{}, &UpgradePlanList{})
}

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

package classifier

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

func TestClassifyPDBThatAllowsDrainAsInformational(t *testing.T) {
	result := Classify([]upgradev1alpha1.Finding{{
		ID:       "PDB_OPERATOR_SAFE",
		Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
		Severity: upgradev1alpha1.RiskLevelCritical,
		Category: "PDB",
		Evidence: []upgradev1alpha1.Evidence{{
			Observed: map[string]string{
				"replicas":     "2",
				"minAvailable": "1",
			},
		}},
	}}, upgradev1alpha1.UpgradeAssessmentSpec{}, time.Now())

	if result.Summary.Blocking != 0 || result.Summary.Informational != 1 {
		t.Fatalf("unexpected classification summary: %#v", result.Summary)
	}
	if got := result.Findings[0].Classification.Status; got != upgradev1alpha1.FindingClassificationInformational {
		t.Fatalf("expected informational classification, got %s", got)
	}
}

func TestClassifyPDBThatBlocksDrainAsBlocking(t *testing.T) {
	result := Classify([]upgradev1alpha1.Finding{{
		ID:       "PDB_OPERATOR_BLOCKING",
		Type:     upgradev1alpha1.FindingTypePDBBlockingRisk,
		Severity: upgradev1alpha1.RiskLevelCritical,
		Category: "PDB",
		Evidence: []upgradev1alpha1.Evidence{{
			Observed: map[string]string{
				"replicas":     "1",
				"minAvailable": "1",
			},
		}},
	}}, upgradev1alpha1.UpgradeAssessmentSpec{}, time.Now())

	if result.Summary.Blocking != 1 {
		t.Fatalf("expected one blocking finding, got %#v", result.Summary)
	}
}

func TestClassifyProviderManagedAKSWebhook(t *testing.T) {
	result := Classify([]upgradev1alpha1.Finding{{
		ID:       "AKS_WEBHOOK",
		Type:     upgradev1alpha1.FindingTypeAdmissionWebhookRisk,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "AdmissionWebhook",
		Resource: &upgradev1alpha1.ResourceRef{
			Kind: "MutatingWebhookConfiguration",
			Name: "aks-node-mutating-webhook",
		},
	}}, upgradev1alpha1.UpgradeAssessmentSpec{}, time.Now())

	if got := result.Findings[0].Classification.Status; got != upgradev1alpha1.FindingClassificationProviderManaged {
		t.Fatalf("expected provider-managed classification, got %s", got)
	}
}

func TestClassifyAcceptedRiskWhenNotExpired(t *testing.T) {
	expires := metav1.NewTime(time.Now().Add(time.Hour))
	result := Classify([]upgradev1alpha1.Finding{{
		ID:       "KNOWN_RISK",
		Type:     upgradev1alpha1.FindingTypeWorkloadAvailability,
		Severity: upgradev1alpha1.RiskLevelHigh,
		Category: "WorkloadAvailability",
	}}, upgradev1alpha1.UpgradeAssessmentSpec{
		AcceptedRisks: []upgradev1alpha1.AcceptedRisk{{
			FindingID:  "KNOWN_RISK",
			Reason:     "Accepted for the upgrade window.",
			ApprovedBy: "platform-team",
			ExpiresAt:  &expires,
		}},
	}, time.Now())

	if got := result.Findings[0].Classification.Status; got != upgradev1alpha1.FindingClassificationAcceptedRisk {
		t.Fatalf("expected accepted-risk classification, got %s", got)
	}
}

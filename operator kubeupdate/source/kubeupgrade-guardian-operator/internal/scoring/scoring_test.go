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

package scoring

import (
	"testing"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

func TestScoreWeightsFindings(t *testing.T) {
	findings := []upgradev1alpha1.Finding{
		{Severity: upgradev1alpha1.RiskLevelCritical},
		{Severity: upgradev1alpha1.RiskLevelHigh},
		{Severity: upgradev1alpha1.RiskLevelMedium},
		{Severity: upgradev1alpha1.RiskLevelLow},
		{Severity: upgradev1alpha1.RiskLevelInfo},
	}

	score, summary := Score(findings)
	if score != 40 {
		t.Fatalf("expected score 40, got %d", score)
	}
	if summary.TotalFindings != 5 || summary.Critical != 1 || summary.High != 1 || summary.Medium != 1 || summary.Low != 1 || summary.Info != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestDecisionOrdering(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		summary  upgradev1alpha1.FindingSummary
		findings []upgradev1alpha1.Finding
		want     upgradev1alpha1.Decision
	}{
		{
			name:    "critical blocks upgrade",
			score:   25,
			summary: upgradev1alpha1.FindingSummary{Critical: 1},
			want:    upgradev1alpha1.DecisionDoNotUpgrade,
		},
		{
			name:    "high score proceeds with caution before incomplete RBAC",
			score:   31,
			summary: upgradev1alpha1.FindingSummary{High: 4},
			findings: []upgradev1alpha1.Finding{{
				Type:     upgradev1alpha1.FindingTypeRBACAssessmentGap,
				Severity: upgradev1alpha1.RiskLevelHigh,
			}},
			want: upgradev1alpha1.DecisionProceedWithCaution,
		},
		{
			name:    "RBAC gap marks assessment incomplete",
			score:   10,
			summary: upgradev1alpha1.FindingSummary{High: 1},
			findings: []upgradev1alpha1.Finding{{
				Type:     upgradev1alpha1.FindingTypeRBACAssessmentGap,
				Severity: upgradev1alpha1.RiskLevelHigh,
			}},
			want: upgradev1alpha1.DecisionAssessmentIncomplete,
		},
		{
			name:    "low score proceeds",
			score:   4,
			summary: upgradev1alpha1.FindingSummary{Medium: 1},
			want:    upgradev1alpha1.DecisionProceed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decision(tt.score, tt.summary, tt.findings)
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestRiskLevelBoundaries(t *testing.T) {
	tests := map[int]upgradev1alpha1.RiskLevel{
		0:  upgradev1alpha1.RiskLevelLow,
		10: upgradev1alpha1.RiskLevelLow,
		11: upgradev1alpha1.RiskLevelMedium,
		30: upgradev1alpha1.RiskLevelMedium,
		31: upgradev1alpha1.RiskLevelHigh,
		60: upgradev1alpha1.RiskLevelHigh,
		61: upgradev1alpha1.RiskLevelCritical,
	}

	for score, want := range tests {
		if got := RiskLevel(score); got != want {
			t.Fatalf("score %d: expected %s, got %s", score, want, got)
		}
	}
}

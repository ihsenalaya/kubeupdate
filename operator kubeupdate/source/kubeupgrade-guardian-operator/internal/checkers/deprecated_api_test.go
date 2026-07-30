package checkers

import (
	"testing"
)

func TestUsesDeprecatedSourceRequiresLastAppliedDeprecatedAPIVersion(t *testing.T) {
	api := removedAPI{APIVersion: "policy/v1beta1", Kind: "PodDisruptionBudget", RemovedIn: 25}

	tests := []struct {
		name       string
		annotation string
		want       bool
	}{
		{
			name:       "legacy source manifest",
			annotation: `{"apiVersion":"policy/v1beta1","kind":"PodDisruptionBudget","metadata":{"name":"legacy"}}`,
			want:       true,
		},
		{
			name:       "modern source manifest served through legacy endpoint",
			annotation: `{"apiVersion":"policy/v1","kind":"PodDisruptionBudget","metadata":{"name":"modern"}}`,
			want:       false,
		},
		{
			name:       "missing source annotation",
			annotation: "",
			want:       false,
		},
		{
			name:       "invalid source annotation",
			annotation: `not-json`,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var annotations map[string]string
			if tt.annotation != "" {
				annotations = map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": tt.annotation,
				}
			}
			if got := usesDeprecatedSource(annotations, api); got != tt.want {
				t.Fatalf("usesDeprecatedSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

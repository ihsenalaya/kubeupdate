package checkers

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
			item := unstructured.Unstructured{}
			if tt.annotation != "" {
				item.SetAnnotations(map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": tt.annotation,
				})
			}
			if got := usesDeprecatedSource(item, api); got != tt.want {
				t.Fatalf("usesDeprecatedSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

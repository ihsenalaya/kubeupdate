package checkers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsBroadNamespaceSelectorTreatsNilAndEmptyAsBroad(t *testing.T) {
	tests := []struct {
		name     string
		selector *metav1.LabelSelector
		want     bool
	}{
		{name: "nil", selector: nil, want: true},
		{name: "empty", selector: &metav1.LabelSelector{}, want: true},
		{name: "match labels", selector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "platform"}}, want: false},
		{
			name: "match expressions",
			selector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "team",
				Operator: metav1.LabelSelectorOpExists,
			}}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBroadNamespaceSelector(tt.selector); got != tt.want {
				t.Fatalf("isBroadNamespaceSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

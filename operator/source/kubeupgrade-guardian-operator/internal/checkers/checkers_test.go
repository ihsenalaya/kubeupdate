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

package checkers

import (
	"context"
	"errors"
	"testing"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

func TestWorkloadAvailabilityDetectsDeploymentReplicasBelowTwo(t *testing.T) {
	ctx := context.Background()
	replicas := int32(1)
	c := fakeClient(
		namespace("production", nil),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "payment-api"}},
				},
			},
		},
	)

	findings, err := WorkloadAvailability{}.Check(ctx, c, assessment("production"))
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypeWorkloadAvailability, upgradev1alpha1.RiskLevelHigh)
	if got := findings[0].Evidence[0].Observed["replicas"]; got != "1" {
		t.Fatalf("expected replicas evidence 1, got %q", got)
	}
}

func TestReadinessProbeDetectsMissingProbe(t *testing.T) {
	ctx := context.Background()
	c := fakeClient(
		namespace("production", nil),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "api", Image: "example/api:latest"}},
					},
				},
			},
		},
	)

	findings, err := ReadinessProbe{}.Check(ctx, c, assessment("production"))
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypeMissingReadinessProbe, upgradev1alpha1.RiskLevelMedium)
	if got := findings[0].Evidence[0].Observed["containerName"]; got != "api" {
		t.Fatalf("expected containerName evidence api, got %q", got)
	}
}

func TestPDBDetectsMinAvailableBlockingSingleReplica(t *testing.T) {
	ctx := context.Background()
	replicas := int32(1)
	minAvailable := intstr.FromInt32(1)
	c := fakeClient(
		namespace("production", nil),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "payment-api"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "payment-api"}},
				},
			},
		},
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: policyv1.PodDisruptionBudgetSpec{
				MinAvailable: &minAvailable,
				Selector:     &metav1.LabelSelector{MatchLabels: map[string]string{"app": "payment-api"}},
			},
		},
	)

	findings, err := PDB{}.Check(ctx, c, assessment("production"))
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypePDBBlockingRisk, upgradev1alpha1.RiskLevelCritical)
	if got := findings[0].Evidence[0].Observed["minAvailable"]; got != "1" {
		t.Fatalf("expected minAvailable evidence 1, got %q", got)
	}
}

func TestAdmissionWebhookDetectsFailPolicyAndMissingService(t *testing.T) {
	ctx := context.Background()
	fail := admissionv1.Fail
	sideEffects := admissionv1.SideEffectClassNone
	c := fakeClient(&admissionv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "policy-webhook"},
		Webhooks: []admissionv1.ValidatingWebhook{{
			Name:          "policy.example.com",
			FailurePolicy: &fail,
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{Namespace: "policy", Name: "missing-webhook"},
			},
			SideEffects:             &sideEffects,
			AdmissionReviewVersions: []string{"v1"},
		}},
	})

	findings, err := AdmissionWebhook{}.Check(ctx, c, &upgradev1alpha1.UpgradeAssessment{})
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypeAdmissionWebhookRisk, upgradev1alpha1.RiskLevelHigh)
	assertFinding(t, findings, upgradev1alpha1.FindingTypeAdmissionWebhookRisk, upgradev1alpha1.RiskLevelCritical)
}

func TestPolicyRiskDetectsRestrictedNamespaceAndPrivilegedWorkload(t *testing.T) {
	ctx := context.Background()
	privileged := true
	c := fakeClient(
		namespace("production", map[string]string{"pod-security.kubernetes.io/enforce": "restricted"}),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            "api",
							Image:           "example/api:latest",
							SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
						}},
					},
				},
			},
		},
	)

	findings, err := PolicyRisk{}.Check(ctx, c, assessment("production"))
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypePolicyRisk, upgradev1alpha1.RiskLevelMedium)
	assertFinding(t, findings, upgradev1alpha1.FindingTypePolicyRisk, upgradev1alpha1.RiskLevelHigh)
}

func TestCapacityDetectsInsufficientOneNodeHeadroom(t *testing.T) {
	ctx := context.Background()
	c := fakeClient(
		node("node-a", "2000m", "4Gi"),
		node("node-b", "2000m", "4Gi"),
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name: "api",
				Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resourceapi.MustParse("3500m"),
					corev1.ResourceMemory: resourceapi.MustParse("2Gi"),
				}},
			}}},
		},
	)

	findings, err := Capacity{}.Check(ctx, c, &upgradev1alpha1.UpgradeAssessment{})
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypeCapacityDrainRisk, upgradev1alpha1.RiskLevelHigh)
}

func TestObservabilityDetectsMissingMonitoringNamespace(t *testing.T) {
	ctx := context.Background()
	c := fakeClient(namespace("production", nil))

	findings, err := Observability{}.Check(ctx, c, &upgradev1alpha1.UpgradeAssessment{})
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypeObservabilityGap, upgradev1alpha1.RiskLevelMedium)
}

func TestRBACGapOnForbiddenList(t *testing.T) {
	ctx := context.Background()
	scheme := testScheme()
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return apierrors.NewForbidden(schema.GroupResource{Group: "", Resource: "namespaces"}, "", errors.New("denied"))
			},
		}).
		Build()

	findings, err := WorkloadAvailability{}.Check(ctx, c, &upgradev1alpha1.UpgradeAssessment{})
	if err != nil {
		t.Fatal(err)
	}
	assertFinding(t, findings, upgradev1alpha1.FindingTypeRBACAssessmentGap, upgradev1alpha1.RiskLevelHigh)
}

func fakeClient(objects ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithObjects(objects...).
		Build()
}

func testScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := admissionv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := upgradev1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return scheme
}

func assessment(namespaces ...string) *upgradev1alpha1.UpgradeAssessment {
	return &upgradev1alpha1.UpgradeAssessment{
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			TargetVersion: "1.32",
			Mode:          upgradev1alpha1.AssessmentModeReadOnly,
			Scope: upgradev1alpha1.AssessmentScope{
				Namespaces: upgradev1alpha1.NamespaceScope{Include: namespaces},
			},
		},
	}
}

func namespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels}}
}

func node(name, cpu, memory string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
			corev1.ResourceCPU:    resourceapi.MustParse(cpu),
			corev1.ResourceMemory: resourceapi.MustParse(memory),
		}},
	}
}

func assertFinding(t *testing.T, findings []upgradev1alpha1.Finding, findingType string, severity upgradev1alpha1.RiskLevel) {
	t.Helper()
	for _, finding := range findings {
		if finding.Type == findingType && finding.Severity == severity {
			return
		}
	}
	t.Fatalf("expected finding type=%s severity=%s in %#v", findingType, severity, findings)
}

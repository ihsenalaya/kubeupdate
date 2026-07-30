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

package snapshot

import (
	"context"
	"errors"
	"testing"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

const productionNamespace = "production"

func TestCollectAppliesNamespaceScopeToWorkloads(t *testing.T) {
	reader := fakeClient(
		namespace(productionNamespace),
		namespace("sandbox"),
		deployment(productionNamespace, "payment-api"),
		deployment("sandbox", "scratch"),
	)

	snap, err := Collect(context.Background(), reader, scope([]string{productionNamespace}, nil))
	if err != nil {
		t.Fatal(err)
	}

	if len(snap.Namespaces) != 1 || snap.Namespaces[0].Name != productionNamespace {
		t.Fatalf("expected only production in scope, got %#v", snap.Namespaces)
	}
	if len(snap.AllNamespaces) != 2 {
		t.Fatalf("expected both namespaces in the unscoped view, got %d", len(snap.AllNamespaces))
	}
	if len(snap.Deployments) != 1 || snap.Deployments[0].Namespace != productionNamespace {
		t.Fatalf("expected only the production deployment, got %#v", snap.Deployments)
	}
}

func TestCollectExcludesNamespaces(t *testing.T) {
	reader := fakeClient(
		namespace(productionNamespace),
		namespace("kube-system"),
		deployment("kube-system", "coredns"),
	)

	snap, err := Collect(context.Background(), reader, scope(nil, []string{"kube-system"}))
	if err != nil {
		t.Fatal(err)
	}

	if len(snap.Namespaces) != 1 || snap.Namespaces[0].Name != productionNamespace {
		t.Fatalf("expected kube-system excluded, got %#v", snap.Namespaces)
	}
	if len(snap.Deployments) != 0 {
		t.Fatalf("expected no deployment retained, got %#v", snap.Deployments)
	}
}

func TestCollectKeepsIncludedNamespaceThatWasNotObserved(t *testing.T) {
	snap, err := Collect(context.Background(), fakeClient(), scope([]string{productionNamespace}, nil))
	if err != nil {
		t.Fatal(err)
	}

	if len(snap.Namespaces) != 1 || snap.Namespaces[0].Name != productionNamespace {
		t.Fatalf("expected the requested namespace to be reported, got %#v", snap.Namespaces)
	}
}

func TestCollectKeepsClusterWideSignalsUnscoped(t *testing.T) {
	reader := fakeClient(
		namespace(productionNamespace),
		namespace("cert-manager"),
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "cert-manager", Name: "cert-manager-0"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "cert-manager", Name: "webhook"}},
	)

	snap, err := Collect(context.Background(), reader, scope([]string{productionNamespace}, nil))
	if err != nil {
		t.Fatal(err)
	}

	if len(snap.Pods) != 1 {
		t.Fatalf("capacity needs every pod, got %#v", snap.Pods)
	}
	if len(snap.Services) != 1 {
		t.Fatalf("webhook backends need every service, got %#v", snap.Services)
	}
}

func TestCollectTurnsForbiddenIntoGapAndKeepsGoing(t *testing.T) {
	reader := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithObjects(namespace(productionNamespace)).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*appsv1.DeploymentList); ok {
					return apierrors.NewForbidden(schema.GroupResource{Group: "apps", Resource: "deployments"}, "", errors.New("denied"))
				}
				return c.List(ctx, list, opts...)
			},
		}).
		Build()

	snap, err := Collect(context.Background(), reader, upgradev1alpha1.AssessmentScope{})
	if err != nil {
		t.Fatalf("an RBAC refusal must not fail collection: %v", err)
	}

	if len(snap.Gaps) != 1 {
		t.Fatalf("expected exactly one gap finding, got %#v", snap.Gaps)
	}
	gap := snap.Gaps[0]
	if gap.Type != upgradev1alpha1.FindingTypeRBACAssessmentGap || gap.Severity != upgradev1alpha1.RiskLevelHigh {
		t.Fatalf("unexpected gap finding %#v", gap)
	}
	if len(gap.Evidence) == 0 || gap.Evidence[0].Observed["resource"] != "deployments" {
		t.Fatalf("gap finding must carry the denied resource as evidence, got %#v", gap.Evidence)
	}
	if len(snap.Namespaces) != 1 {
		t.Fatalf("collection must continue after a denied type, got %#v", snap.Namespaces)
	}
}

func TestCollectPropagatesNonRBACErrors(t *testing.T) {
	reader := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return errors.New("api server unreachable")
			},
		}).
		Build()

	if _, err := Collect(context.Background(), reader, upgradev1alpha1.AssessmentScope{}); err == nil {
		t.Fatal("expected a non-RBAC error to abort collection")
	}
}

func TestCollectPagesThroughContinueTokens(t *testing.T) {
	pages := 0
	reader := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				nodes, ok := list.(*corev1.NodeList)
				if !ok {
					return c.List(ctx, list, opts...)
				}
				pages++
				nodes.Items = []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "node"}}}
				if pages < 3 {
					nodes.Continue = "next"
					return nil
				}
				nodes.Continue = ""
				return nil
			},
		}).
		Build()

	snap, err := Collect(context.Background(), reader, upgradev1alpha1.AssessmentScope{})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 3 {
		t.Fatalf("expected 3 pages to be followed, got %d", pages)
	}
	if len(snap.Nodes) != 3 {
		t.Fatalf("expected every page to be accumulated, got %d nodes", len(snap.Nodes))
	}
}

func TestCollectIndexesCRDNames(t *testing.T) {
	reader := fakeClient(&apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "prometheuses.monitoring.coreos.com"},
	})

	snap, err := Collect(context.Background(), reader, upgradev1alpha1.AssessmentScope{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snap.CRDNames["prometheuses.monitoring.coreos.com"]; !ok {
		t.Fatalf("expected the CRD to be indexed, got %#v", snap.CRDNames)
	}
}

func fakeClient(objects ...client.Object) client.Reader {
	return fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(objects...).Build()
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

func scope(include, exclude []string) upgradev1alpha1.AssessmentScope {
	return upgradev1alpha1.AssessmentScope{
		Namespaces: upgradev1alpha1.NamespaceScope{Include: include, Exclude: exclude},
	}
}

func namespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func deployment(namespace, name string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
}

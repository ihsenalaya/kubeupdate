package checkers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
)

func TestR10FixtureRunner(t *testing.T) {
	fixtureDir := os.Getenv("R10_FIXTURE_DIR")
	outputPath := os.Getenv("R10_OUTPUT")
	if fixtureDir == "" || outputPath == "" {
		t.Skip("set R10_FIXTURE_DIR and R10_OUTPUT to run the R10 fixture harness")
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := upgradev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	addLegacyTypes(scheme)

	objects, err := loadFixtureObjects(scheme, fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	objects = append(objects, syntheticNodes()...)
	objects = append(objects, syntheticPods(objects)...)

	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	assessment := &upgradev1alpha1.UpgradeAssessment{
		ObjectMeta: metav1.ObjectMeta{Name: "r10-assessment", Namespace: "default"},
		Spec: upgradev1alpha1.UpgradeAssessmentSpec{
			TargetVersion: "1.35",
		},
	}

	var findings []upgradev1alpha1.Finding
	for _, checker := range Default(assessment) {
		checkerFindings, err := checker.Check(context.Background(), kubeClient, assessment)
		if err != nil {
			t.Fatalf("%s: %v", checker.Name(), err)
		}
		findings = append(findings, checkerFindings...)
	}

	sort.Slice(findings, func(i, j int) bool {
		left := findingSortKey(findings[i])
		right := findingSortKey(findings[j])
		return left < right
	})

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(map[string]any{"findings": findings}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func addLegacyTypes(scheme *runtime.Scheme) {
	for _, gvk := range []schema.GroupVersionKind{
		{Group: "batch", Version: "v1beta1", Kind: "CronJob"},
		{Group: "networking.k8s.io", Version: "v1beta1", Kind: "Ingress"},
		{Group: "policy", Version: "v1beta1", Kind: "PodDisruptionBudget"},
		{Group: "policy", Version: "v1beta1", Kind: "PodSecurityPolicy"},
		{Group: "autoscaling", Version: "v2beta2", Kind: "HorizontalPodAutoscaler"},
	} {
		if !scheme.Recognizes(gvk) {
			scheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		}
		listGVK := schema.GroupVersionKind{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind + "List",
		}
		if !scheme.Recognizes(listGVK) {
			scheme.AddKnownTypeWithName(listGVK, &unstructured.UnstructuredList{})
		}
	}
}

func loadFixtureObjects(scheme *runtime.Scheme, fixtureDir string) ([]client.Object, error) {
	paths, err := filepath.Glob(filepath.Join(fixtureDir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	var objects []client.Object
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		decoder := yaml.NewYAMLOrJSONDecoder(file, 4096)
		for {
			var raw map[string]any
			if err := decoder.Decode(&raw); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				file.Close()
				return nil, err
			}
			if len(raw) == 0 {
				continue
			}
			object, err := fixtureObject(scheme, raw)
			if err != nil {
				file.Close()
				return nil, err
			}
			objects = append(objects, object)
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	return objects, nil
}

func fixtureObject(scheme *runtime.Scheme, raw map[string]any) (client.Object, error) {
	obj := &unstructured.Unstructured{Object: raw}
	addLastAppliedAnnotation(obj)

	gvk := obj.GroupVersionKind()
	typed, err := scheme.New(gvk)
	if err != nil {
		return obj, nil
	}
	if unstructuredObj, ok := typed.(*unstructured.Unstructured); ok {
		unstructuredObj.SetUnstructuredContent(obj.Object)
		return unstructuredObj, nil
	}
	target, ok := typed.(client.Object)
	if !ok {
		return obj, nil
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, target); err != nil {
		return nil, err
	}
	return target, nil
}

func addLastAppliedAnnotation(obj *unstructured.Unstructured) {
	gvk := obj.GroupVersionKind()
	switch gvk {
	case schema.GroupVersionKind{Group: "batch", Version: "v1beta1", Kind: "CronJob"},
		schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1beta1", Kind: "Ingress"},
		schema.GroupVersionKind{Group: "policy", Version: "v1beta1", Kind: "PodDisruptionBudget"},
		schema.GroupVersionKind{Group: "policy", Version: "v1beta1", Kind: "PodSecurityPolicy"},
		schema.GroupVersionKind{Group: "autoscaling", Version: "v2beta2", Kind: "HorizontalPodAutoscaler"}:
	default:
		return
	}
	header, _ := json.Marshal(map[string]string{
		"apiVersion": obj.GetAPIVersion(),
		"kind":       obj.GetKind(),
	})
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(header)
	obj.SetAnnotations(annotations)
}

func syntheticNodes() []client.Object {
	var nodes []client.Object
	for i := 1; i <= 3; i++ {
		nodes = append(nodes, &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "r10-node-" + string(rune('0'+i))},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    apiresource.MustParse("2"),
					corev1.ResourceMemory: apiresource.MustParse("4Gi"),
				},
			},
		})
	}
	return nodes
}

func syntheticPods(objects []client.Object) []client.Object {
	var pods []client.Object
	for _, object := range objects {
		switch item := object.(type) {
		case *appsv1.Deployment:
			replicas := int32(1)
			if item.Spec.Replicas != nil {
				replicas = *item.Spec.Replicas
			}
			for i := int32(0); i < replicas; i++ {
				pods = append(pods, syntheticPod("Deployment", item.Namespace, item.Name, item.Spec.Template.Labels, item.Spec.Template.Spec, i))
			}
		case *appsv1.StatefulSet:
			replicas := int32(1)
			if item.Spec.Replicas != nil {
				replicas = *item.Spec.Replicas
			}
			for i := int32(0); i < replicas; i++ {
				pods = append(pods, syntheticPod("StatefulSet", item.Namespace, item.Name, item.Spec.Template.Labels, item.Spec.Template.Spec, i))
			}
		}
	}
	return pods
}

func syntheticPod(kind, namespace, ownerName string, labels map[string]string, spec corev1.PodSpec, index int32) client.Object {
	controller := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ownerName + "-synthetic-" + string(rune('0'+index)),
			Namespace: namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       kind,
				Name:       ownerName,
				Controller: &controller,
			}},
		},
		Spec: spec,
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
}

func findingSortKey(finding upgradev1alpha1.Finding) string {
	resource := finding.Resource
	if resource == nil {
		return finding.Category + "/" + string(finding.Type) + "/" + finding.Message
	}
	return finding.Category + "/" + string(finding.Type) + "/" + resource.Kind + "/" + resource.Namespace + "/" + resource.Name + "/" + finding.Message
}

package checkers

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	upgradev1alpha1 "github.com/ihsenalaya/kubeupgrade-guardian-operator/api/v1alpha1"
	"github.com/ihsenalaya/kubeupgrade-guardian-operator/internal/snapshot"
)

func TestCapacityExcludesPodsThatNeverReschedule(t *testing.T) {
	controller := true
	snap := &snapshot.ClusterSnapshot{
		Nodes: []corev1.Node{node("node-a", "2000m", "4Gi"), node("node-b", "2000m", "4Gi")},
		Pods: []corev1.Pod{
			podWithRequests("daemon", "3500m", "2Gi", func(pod *corev1.Pod) {
				pod.OwnerReferences = []metav1.OwnerReference{{
					Kind: "DaemonSet", Name: "node-agent", Controller: &controller,
				}}
			}),
			podWithRequests("static", "3500m", "2Gi", func(pod *corev1.Pod) {
				pod.Annotations = map[string]string{mirrorPodAnnotation: "abc123"}
			}),
			podWithRequests("finished", "3500m", "2Gi", func(pod *corev1.Pod) {
				pod.Status.Phase = corev1.PodSucceeded
			}),
		},
	}

	if findings := (Capacity{}).Check(snap, &upgradev1alpha1.UpgradeAssessment{}); len(findings) != 0 {
		t.Fatalf("DaemonSet, static and terminated pods do not have to be rescheduled, got %#v", findings)
	}
}

func TestCapacityCountsDemandOfReschedulablePodsOnly(t *testing.T) {
	controller := true
	// Asymmetric nodes: losing the largest leaves 2000m of CPU headroom.
	snap := &snapshot.ClusterSnapshot{
		Nodes: []corev1.Node{node("node-a", "2000m", "4Gi"), node("node-b", "3000m", "6Gi")},
		Pods: []corev1.Pod{
			podWithRequests("payment-api", "1700m", "1Gi", nil),
			podWithRequests("daemon", "1000m", "1Gi", func(pod *corev1.Pod) {
				pod.OwnerReferences = []metav1.OwnerReference{{
					Kind: "DaemonSet", Name: "node-agent", Controller: &controller,
				}}
			}),
		},
	}

	findings := Capacity{}.Check(snap, &upgradev1alpha1.UpgradeAssessment{})
	if len(findings) != 1 {
		t.Fatalf("1700m of demand against 2000m of headroom is above 80%%, expected a finding, got %#v", findings)
	}
	observed := findings[0].Evidence[0].Observed
	if observed["requestedCPU"] != "1700m" {
		t.Fatalf("expected only the reschedulable demand, got %q", observed["requestedCPU"])
	}
	if observed["reschedulablePods"] != "1" || observed["nonReschedulablePods"] != "1" {
		t.Fatalf("expected the pod split in evidence, got %#v", observed)
	}
}

func TestEffectivePodRequestsUsesTheLargestInitContainer(t *testing.T) {
	spec := corev1.PodSpec{
		InitContainers: []corev1.Container{
			container("migrate", "2000m", "1Gi"),
			container("seed", "500m", "256Mi"),
		},
		Containers: []corev1.Container{container("api", "500m", "512Mi")},
	}

	cpu, memory := effectivePodRequests(spec)
	if cpu != 2000 {
		t.Fatalf("expected the largest init container to dominate, got %dm", cpu)
	}
	if memory != 1024*1024*1024 {
		t.Fatalf("expected 1Gi of memory, got %d", memory)
	}
}

func TestEffectivePodRequestsAddsSidecarsToTheRegularContainers(t *testing.T) {
	always := corev1.ContainerRestartPolicyAlways
	sidecar := container("proxy", "500m", "256Mi")
	sidecar.RestartPolicy = &always

	spec := corev1.PodSpec{
		InitContainers: []corev1.Container{sidecar, container("migrate", "800m", "128Mi")},
		Containers:     []corev1.Container{container("api", "500m", "512Mi")},
	}

	cpu, _ := effectivePodRequests(spec)
	if cpu != 1000 {
		t.Fatalf("a restartPolicy=Always init container runs for the whole pod lifetime and must be summed, got %dm", cpu)
	}
}

func TestEffectivePodRequestsAddsPodOverhead(t *testing.T) {
	spec := corev1.PodSpec{
		Containers: []corev1.Container{container("api", "500m", "512Mi")},
		Overhead: corev1.ResourceList{
			corev1.ResourceCPU:    resourceapi.MustParse("250m"),
			corev1.ResourceMemory: resourceapi.MustParse("128Mi"),
		},
	}

	cpu, memory := effectivePodRequests(spec)
	if cpu != 750 {
		t.Fatalf("expected the runtime class overhead to be added, got %dm", cpu)
	}
	if memory != 640*1024*1024 {
		t.Fatalf("expected 640Mi of memory, got %d", memory)
	}
}

func container(name, cpu, memory string) corev1.Container {
	return corev1.Container{
		Name: name,
		Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resourceapi.MustParse(cpu),
			corev1.ResourceMemory: resourceapi.MustParse(memory),
		}},
	}
}

func podWithRequests(name, cpu, memory string, mutate func(*corev1.Pod)) corev1.Pod {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "production"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{container("app", cpu, memory)}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if mutate != nil {
		mutate(&pod)
	}
	return pod
}

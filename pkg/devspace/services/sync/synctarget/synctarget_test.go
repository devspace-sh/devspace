package synctarget

import (
	"context"
	"testing"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	kubeselector "github.com/loft-sh/devspace/pkg/devspace/kubectl/selector"
	kubectesting "github.com/loft-sh/devspace/pkg/devspace/kubectl/testing"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	"github.com/loft-sh/devspace/pkg/util/log"
	fakelogger "github.com/loft-sh/devspace/pkg/util/log/testing"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"gotest.tools/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type fakeTargetSelector struct {
	out *kubeselector.SelectedPodContainer
	err error
}

func (f *fakeTargetSelector) SelectSinglePod(ctx context.Context, client kubectl.Client, lg log.Logger) (*corev1.Pod, error) {
	panic("not implemented")
}

func (f *fakeTargetSelector) SelectSingleContainer(ctx context.Context, client kubectl.Client, lg log.Logger) (*kubeselector.SelectedPodContainer, error) {
	return f.out, f.err
}

func (f *fakeTargetSelector) WithContainer(container string) targetselector.TargetSelector {
	return f
}

func testContainer(name string) *corev1.Container {
	return &corev1.Container{Name: name, Image: "test:latest"}
}

func TestConfigForIndex_PrimaryIsBiDirectional(t *testing.T) {
	base := &latest.SyncConfig{
		DisableDownload: false,
		DisableUpload:   true,
		OnUpload:        &latest.SyncOnUpload{RestartContainer: true},
	}
	out := ConfigForIndex(base, 0)
	assert.Equal(t, false, out.DisableDownload)
	assert.Equal(t, true, out.DisableUpload)
	assert.Assert(t, out.OnUpload != nil)
	assert.Equal(t, true, out.OnUpload.RestartContainer)
}

func TestConfigForIndex_SecondaryIsUploadOnly(t *testing.T) {
	base := &latest.SyncConfig{
		DisableDownload: false,
		DisableUpload:   false,
		OnUpload:        &latest.SyncOnUpload{RestartContainer: true},
	}
	out := ConfigForIndex(base, 1)
	assert.Equal(t, true, out.DisableDownload)
	assert.Equal(t, false, out.DisableUpload)
	assert.Assert(t, out.OnUpload == nil)
}

func TestBuildTargets_SyncReplicasOff_NoExpansion(t *testing.T) {
	ctx := context.Background()
	lg := fakelogger.NewFakeLogger()
	kube := fake.NewSimpleClientset()
	client := kubectesting.Client{Client: kube}
	sel := targetselector.NewTargetSelector(targetselector.NewEmptyOptions())
	cfg := &latest.SyncConfig{SyncReplicas: false, Path: "./:/app"}

	targets, err := BuildTargets(ctx, lg, &client, sel, cfg)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(targets))
	assert.Equal(t, "", targets[0].Pod)
}

func TestBuildTargets_SyncReplicasOn_OrdersNewestPodFirst(t *testing.T) {
	ctx := context.Background()
	lg := fakelogger.NewFakeLogger()
	ns := "default"

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "img:v1"}},
				},
			},
		},
	}

	kube := fake.NewSimpleClientset(deploy)
	dep, err := kube.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
	assert.NilError(t, err)

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-rs",
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "web",
				UID:        dep.UID,
				Controller: ptr.Bool(true),
			}},
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: deploy.Spec.Selector,
			Template: deploy.Spec.Template,
		},
	}
	rs, err = kube.AppsV1().ReplicaSets(ns).Create(ctx, rs, metav1.CreateOptions{})
	assert.NilError(t, err)

	oldTime := metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	newTime := metav1.NewTime(time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC))

	podOlder := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "web-older",
			Namespace:         ns,
			Labels:            map[string]string{"app": "web"},
			CreationTimestamp: oldTime,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       rs.Name,
				UID:        rs.UID,
				Controller: ptr.Bool(true),
			}},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "img:v1"}},
		},
	}
	podNewer := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "web-newer",
			Namespace:         ns,
			Labels:            map[string]string{"app": "web"},
			CreationTimestamp: newTime,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       rs.Name,
				UID:        rs.UID,
				Controller: ptr.Bool(true),
			}},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "img:v1"}},
		},
	}
	_, err = kube.CoreV1().Pods(ns).Create(ctx, podOlder, metav1.CreateOptions{})
	assert.NilError(t, err)
	_, err = kube.CoreV1().Pods(ns).Create(ctx, podNewer, metav1.CreateOptions{})
	assert.NilError(t, err)

	primary := &kubeselector.SelectedPodContainer{
		Pod:       podNewer,
		Container: testContainer("app"),
	}
	sel := &fakeTargetSelector{out: primary}
	cfg := &latest.SyncConfig{SyncReplicas: true, Path: "./:/app"}

	client := kubectesting.Client{Client: kube}
	targets, err := BuildTargets(ctx, lg, &client, sel, cfg)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(targets))
	assert.Equal(t, "web-newer", targets[0].Pod)
	assert.Equal(t, "web-older", targets[1].Pod)
}

func TestBuildTargets_FiltersPodsFromOtherDeploymentsWithSameLabels(t *testing.T) {
	ctx := context.Background()
	lg := fakelogger.NewFakeLogger()
	ns := "default"
	label := map[string]string{"app": "shared"}

	deployA := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-a", Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: label},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: label},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "a"}}},
			},
		},
	}
	deployB := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-b", Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: label},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: label},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "b"}}},
			},
		},
	}
	kube := fake.NewSimpleClientset(deployA, deployB)

	depA, err := kube.AppsV1().Deployments(ns).Get(ctx, "svc-a", metav1.GetOptions{})
	assert.NilError(t, err)
	depB, err := kube.AppsV1().Deployments(ns).Get(ctx, "svc-b", metav1.GetOptions{})
	assert.NilError(t, err)

	rsA := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "svc-a-rs", Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "Deployment", Name: "svc-a", UID: depA.UID, Controller: ptr.Bool(true),
			}},
		},
		Spec: appsv1.ReplicaSetSpec{Selector: deployA.Spec.Selector, Template: deployA.Spec.Template},
	}
	rsB := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "svc-b-rs", Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "Deployment", Name: "svc-b", UID: depB.UID, Controller: ptr.Bool(true),
			}},
		},
		Spec: appsv1.ReplicaSetSpec{Selector: deployB.Spec.Selector, Template: deployB.Spec.Template},
	}
	rsA, err = kube.AppsV1().ReplicaSets(ns).Create(ctx, rsA, metav1.CreateOptions{})
	assert.NilError(t, err)
	rsB, err = kube.AppsV1().ReplicaSets(ns).Create(ctx, rsB, metav1.CreateOptions{})
	assert.NilError(t, err)

	podA := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-a", Namespace: ns, Labels: label,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "ReplicaSet", Name: rsA.Name, UID: rsA.UID, Controller: ptr.Bool(true),
			}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "a"}}},
	}
	podB := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-b", Namespace: ns, Labels: label,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "ReplicaSet", Name: rsB.Name, UID: rsB.UID, Controller: ptr.Bool(true),
			}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "b"}}},
	}
	_, err = kube.CoreV1().Pods(ns).Create(ctx, podA, metav1.CreateOptions{})
	assert.NilError(t, err)
	_, err = kube.CoreV1().Pods(ns).Create(ctx, podB, metav1.CreateOptions{})
	assert.NilError(t, err)

	primary := &kubeselector.SelectedPodContainer{Pod: podA, Container: testContainer("app")}
	sel := &fakeTargetSelector{out: primary}
	cfg := &latest.SyncConfig{SyncReplicas: true, Path: "./:/app"}

	client := kubectesting.Client{Client: kube}
	targets, err := BuildTargets(ctx, lg, &client, sel, cfg)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(targets))
	assert.Equal(t, "pod-a", targets[0].Pod)
}

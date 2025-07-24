/*
Copyright 2025.

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

package controller

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestConfigMapReplication(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConfigMap Reconciler Suite")
}

var _ = Describe("ConfigMap Replication", func() {
	ctx := context.Background()

	It("should replicate configmap to specified namespaces", func() {
		// Create test namespaces
		ns1 := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cm-test-ns1"}}
		ns2 := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cm-test-ns2"}}
		Expect(k8sClient.Create(ctx, ns1)).To(Succeed())
		Expect(k8sClient.Create(ctx, ns2)).To(Succeed())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "replicated-config",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					replicateKeyCM: "cm-test-ns2",
				},
			},
			Data: map[string]string{"app.conf": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		// Wait for configmap to be created in source namespace
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns1.Name}, &corev1.ConfigMap{})
		}, 30*time.Second).Should(Succeed())

		// Wait for configmap to be replicated to target namespace
		Eventually(func() error {
			var replicated corev1.ConfigMap
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &replicated)
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

	It("should trigger rollout if configmap used in deployment", func() {
		// Create test namespace
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cm-test-rollout"}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rollout-config",
				Namespace: ns.Name,
				Annotations: map[string]string{
					replicateKeyCM:       ns.Name,
					rolloutOnUpdateKeyCM: "true",
				},
			},
			Data: map[string]string{"config": "val"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		// Wait for configmap to be created
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns.Name}, &corev1.ConfigMap{})
		}, 30*time.Second).Should(Succeed())

		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "rollout-deploy", Namespace: ns.Name},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointerTo[int32](1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:    "main",
							Image:   "busybox",
							Command: []string{"sleep", "3600"},
							EnvFrom: []corev1.EnvFromSource{{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "rollout-config",
									},
								},
							}},
						}},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, deploy)).To(Succeed())

		// Wait for deployment to be created
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &appsv1.Deployment{})
		}, 30*time.Second).Should(Succeed())

		// Patch ConfigMap to trigger the controller
		patch := client.MergeFrom(cm.DeepCopy())
		cm.Data["config"] = "updated-val"
		Expect(k8sClient.Patch(ctx, cm, patch)).To(Succeed())

		Eventually(func() string {
			var d appsv1.Deployment
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &d)
			return d.Spec.Template.Annotations["configmap.restartedAt"]
		}, 30*time.Second, 1*time.Second).ShouldNot(BeEmpty())
	})
})

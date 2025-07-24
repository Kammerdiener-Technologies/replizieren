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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSecretReplication(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secret Reconciler Suite")
}

var _ = Describe("Secret Replication", func() {
	ctx := context.Background()
	var namespace1, namespace2 *corev1.Namespace

	BeforeEach(func() {
		// Create new namespaces
		namespace1 = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns1"}}
		namespace2 = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns2"}}
		Expect(k8sClient.Create(ctx, namespace1)).To(Succeed())
		Expect(k8sClient.Create(ctx, namespace2)).To(Succeed())

		// Wait for namespaces to be ready
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: namespace1.Name}, &corev1.Namespace{})
		}, 30*time.Second).Should(Succeed())
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: namespace2.Name}, &corev1.Namespace{})
		}, 30*time.Second).Should(Succeed())
	})

	AfterEach(func() {
		// Delete namespaces if they exist
		ns1Delete := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns1"}}
		ns2Delete := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns2"}}
		_ = k8sClient.Delete(ctx, ns1Delete)
		_ = k8sClient.Delete(ctx, ns2Delete)

		// Wait for namespaces to be deleted
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: "test-ns1"}, &corev1.Namespace{})
		}, 30*time.Second).ShouldNot(Succeed())
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: "test-ns2"}, &corev1.Namespace{})
		}, 30*time.Second).ShouldNot(Succeed())
	})

	It("should replicate secret to listed namespaces", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "replicated-secret",
				Namespace: namespace1.Name,
				Annotations: map[string]string{
					replicateKeyS: namespace2.Name,
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Wait for secret to be created in source namespace
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: namespace1.Name}, &corev1.Secret{})
		}, 30*time.Second).Should(Succeed())

		// Check replication in ns2
		Eventually(func() error {
			var replicated corev1.Secret
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: namespace2.Name}, &replicated)
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

	It("should trigger rollout if secret used in deployment", func() {
		// Create secret with rollout enabled
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rollout-secret",
				Namespace: namespace1.Name,
				Annotations: map[string]string{
					replicateKeyS:       namespace1.Name,
					rolloutOnUpdateKeyS: "true",
				},
			},
			StringData: map[string]string{"token": "abc"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Wait for secret to be created
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: namespace1.Name}, &corev1.Secret{})
		}, 30*time.Second).Should(Succeed())

		// Create a deployment that uses the secret
		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rollout-deploy",
				Namespace: namespace1.Name,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointerTo[int32](1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "rollout",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "rollout",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test",
								Image: "busybox",
								EnvFrom: []corev1.EnvFromSource{
									{
										SecretRef: &corev1.SecretEnvSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "rollout-secret",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, deploy)).To(Succeed())

		// Wait for deployment to be created
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: namespace1.Name}, &appsv1.Deployment{})
		}, 30*time.Second).Should(Succeed())

		// Trigger update
		patch := client.MergeFrom(secret.DeepCopy())
		secret.Data = map[string][]byte{"token": []byte("updated")}
		Expect(k8sClient.Patch(ctx, secret, patch)).To(Succeed())

		// Ensure annotation is updated (rollout triggered)
		Eventually(func() string {
			var d appsv1.Deployment
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: namespace1.Name}, &d)
			return d.Spec.Template.Annotations["secret.restartedAt"]
		}, 30*time.Second, 1*time.Second).ShouldNot(BeEmpty())
	})
})

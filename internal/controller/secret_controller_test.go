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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Secret Replication", func() {
	const (
		timeout  = 30 * time.Second
		interval = 1 * time.Second
	)

	// Test 1: Basic single namespace replication
	It("should replicate secret to single namespace", func() {
		ns1 := createNamespace("s-single-src")
		ns2 := createNamespace("s-single-tgt")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "single-ns-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name,
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 2: Multiple namespace replication
	It("should replicate secret to multiple namespaces", func() {
		ns1 := createNamespace("s-multi-src")
		ns2 := createNamespace("s-multi-tgt1")
		ns3 := createNamespace("s-multi-tgt2")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multi-ns-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name + "," + ns3.Name,
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns3.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 3: Replicate to all namespaces
	It("should replicate secret to all namespaces when annotation is true", func() {
		ns1 := createNamespace("s-all-src")
		ns2 := createNamespace("s-all-tgt1")
		ns3 := createNamespace("s-all-tgt2")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "all-ns-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: "true",
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Should replicate to both target namespaces
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns3.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 4: Skip replication when annotation is empty
	It("should skip replication when annotation is empty", func() {
		ns1 := createNamespace("s-empty-src")
		ns2 := createNamespace("s-empty-tgt")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-replicate-secret",
				Namespace: ns1.Name,
				// No ReplicateKey annotation
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Give it some time to potentially replicate
		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, 5*time.Second, interval).ShouldNot(Succeed())
	})

	// Test 5: Skip replication when annotation is "false"
	It("should skip replication when annotation is false", func() {
		ns1 := createNamespace("s-false-src")
		ns2 := createNamespace("s-false-tgt")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "false-replicate-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: "false",
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, 5*time.Second, interval).ShouldNot(Succeed())
	})

	// Test 6: Update existing secret in target namespace
	It("should update existing secret in target namespace", func() {
		ns1 := createNamespace("s-update-src")
		ns2 := createNamespace("s-update-tgt")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "update-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name,
				},
			},
			StringData: map[string]string{"key": "original"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Wait for initial replication
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())

		// Update the secret
		patch := client.MergeFrom(secret.DeepCopy())
		secret.StringData = map[string]string{"key": "updated"}
		Expect(k8sClient.Patch(ctx, secret, patch)).To(Succeed())

		// Verify update was replicated
		Eventually(func() string {
			var replicated corev1.Secret
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &replicated); err != nil {
				return ""
			}
			return string(replicated.Data["key"])
		}, timeout, interval).Should(Equal("updated"))
	})

	// Test 7: Handle whitespace in namespace list
	It("should handle whitespace in namespace list", func() {
		ns1 := createNamespace("s-ws-src")
		ns2 := createNamespace("s-ws-tgt")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "whitespace-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: "  " + ns2.Name + "  ", // Leading/trailing whitespace
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 8: Handle empty entries in namespace list
	It("should handle empty entries in namespace list", func() {
		ns1 := createNamespace("s-empty-entry-src")
		ns2 := createNamespace("s-empty-entry-tgt")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "empty-entry-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name + ",,", // Empty entries
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 9: Exclude source namespace from replication targets
	It("should exclude source namespace from replication targets", func() {
		ns1 := createNamespace("s-exclude-src")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "exclude-self-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns1.Name, // Target is same as source
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Verify only one secret exists in source namespace
		Consistently(func() int {
			var secrets corev1.SecretList
			if err := k8sClient.List(ctx, &secrets, client.InNamespace(ns1.Name)); err != nil {
				return -1
			}
			count := 0
			for _, s := range secrets.Items {
				if s.Name == secret.Name {
					count++
				}
			}
			return count
		}, 5*time.Second, interval).Should(Equal(1))
	})

	// Test 10: Trigger rollout when annotation is true
	It("should trigger rollout when rollout-on-update annotation is true", func() {
		ns := createNamespace("s-rollout-ns")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rollout-secret",
				Namespace: ns.Name,
				Annotations: map[string]string{
					ReplicateKey:       ns.Name,
					RolloutOnUpdateKey: "true",
				},
			},
			StringData: map[string]string{"token": "abc"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		deploy := createDeploymentWithSecretEnvFrom(ns.Name, "rollout-deploy", secret.Name)
		Expect(k8sClient.Create(ctx, deploy)).To(Succeed())

		// Wait for deployment to be created
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &appsv1.Deployment{})
		}, timeout, interval).Should(Succeed())

		// Trigger update
		patch := client.MergeFrom(secret.DeepCopy())
		secret.StringData = map[string]string{"token": "updated"}
		Expect(k8sClient.Patch(ctx, secret, patch)).To(Succeed())

		// Verify rollout annotation is set
		Eventually(func() string {
			var d appsv1.Deployment
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &d); err != nil {
				return ""
			}
			if d.Spec.Template.Annotations == nil {
				return ""
			}
			return d.Spec.Template.Annotations["secret.restartedAt"]
		}, timeout, interval).ShouldNot(BeEmpty())
	})

	// Test 11: Replicate data matches source
	It("should replicate secret data that matches source", func() {
		ns1 := createNamespace("s-data-src")
		ns2 := createNamespace("s-data-tgt")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "data-match-secret",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name,
				},
			},
			StringData: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			Type: corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		Eventually(func() map[string][]byte {
			var replicated corev1.Secret
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: ns2.Name}, &replicated); err != nil {
				return nil
			}
			return replicated.Data
		}, timeout, interval).Should(And(
			HaveKeyWithValue("username", []byte("admin")),
			HaveKeyWithValue("password", []byte("secret123")),
		))
	})

	// Test 12: Deployment with volume mount triggers rollout
	It("should trigger rollout for deployment using secret as volume", func() {
		ns := createNamespace("s-vol-rollout-ns")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vol-rollout-secret",
				Namespace: ns.Name,
				Annotations: map[string]string{
					ReplicateKey:       ns.Name,
					RolloutOnUpdateKey: "true",
				},
			},
			StringData: map[string]string{"config": "data"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		deploy := createDeploymentWithSecretVolume(ns.Name, "vol-deploy", secret.Name)
		Expect(k8sClient.Create(ctx, deploy)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &appsv1.Deployment{})
		}, timeout, interval).Should(Succeed())

		// Trigger update
		patch := client.MergeFrom(secret.DeepCopy())
		secret.StringData = map[string]string{"config": "updated-data"}
		Expect(k8sClient.Patch(ctx, secret, patch)).To(Succeed())

		Eventually(func() string {
			var d appsv1.Deployment
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &d); err != nil {
				return ""
			}
			if d.Spec.Template.Annotations == nil {
				return ""
			}
			return d.Spec.Template.Annotations["secret.restartedAt"]
		}, timeout, interval).ShouldNot(BeEmpty())
	})
})

// Helper functions

func createNamespace(name string) *corev1.Namespace {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns
}

func createDeploymentWithSecretEnvFrom(namespace, name, secretName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointerTo[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "main",
						Image: "busybox",
						EnvFrom: []corev1.EnvFromSource{{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
							},
						}},
					}},
				},
			},
		},
	}
}

func createDeploymentWithSecretVolume(namespace, name, secretName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointerTo[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "main",
						Image: "busybox",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "secret-vol",
							MountPath: "/secrets",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "secret-vol",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: secretName,
							},
						},
					}},
				},
			},
		},
	}
}

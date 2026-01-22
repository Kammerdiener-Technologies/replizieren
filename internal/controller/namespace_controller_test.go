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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Namespace Controller", func() {
	const (
		timeout  = 30 * time.Second
		interval = 1 * time.Second
	)

	// Test 1: New namespace should trigger replication of replicate-all secrets
	It("should replicate secrets with replicate-all to new namespace", func() {
		srcNs := createTestNamespace("ns-src-secret")

		// Create a secret with replicate-all annotation
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ns-watch-secret",
				Namespace: srcNs.Name,
				Annotations: map[string]string{
					ReplicateAllKey: "true",
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Create a new namespace - the secret should be replicated to it
		targetNs := createTestNamespace("ns-tgt-secret")

		// Verify the secret was replicated
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: targetNs.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 2: New namespace should trigger replication of replicate-all configmaps
	It("should replicate configmaps with replicate-all to new namespace", func() {
		srcNs := createTestNamespace("ns-src-cm")

		// Create a configmap with replicate-all annotation
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ns-watch-configmap",
				Namespace: srcNs.Name,
				Annotations: map[string]string{
					ReplicateAllKey: "true",
				},
			},
			Data: map[string]string{"config": "data"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		// Create a new namespace - the configmap should be replicated to it
		targetNs := createTestNamespace("ns-tgt-cm")

		// Verify the configmap was replicated
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: targetNs.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 3: Secrets with specific namespace targets should not replicate to unrelated namespaces
	It("should not replicate secrets with specific targets to unrelated namespaces", func() {
		srcNs := createTestNamespace("ns-src-specific")
		specificTarget := createTestNamespace("ns-specific-target")

		// Create a secret with specific namespace target
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "specific-target-secret",
				Namespace: srcNs.Name,
				Annotations: map[string]string{
					ReplicateKey: specificTarget.Name,
				},
			},
			StringData: map[string]string{"key": "value"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Wait for replication to specific target
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: specificTarget.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())

		// Create an unrelated namespace
		unrelatedNs := createTestNamespace("ns-unrelated")

		// Secret should NOT be replicated to the unrelated namespace
		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: unrelatedNs.Name}, &corev1.Secret{})
		}, 5*time.Second, interval).ShouldNot(Succeed())
	})

	// Test 4: Replicated data should match source
	It("should replicate secret data that matches source to new namespace", func() {
		srcNs := createTestNamespace("ns-src-data")

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "data-match-ns-secret",
				Namespace: srcNs.Name,
				Annotations: map[string]string{
					ReplicateAllKey: "true",
				},
			},
			StringData: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			Type: corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		targetNs := createTestNamespace("ns-tgt-data")

		Eventually(func() map[string][]byte {
			var replicated corev1.Secret
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: targetNs.Name}, &replicated); err != nil {
				return nil
			}
			return replicated.Data
		}, timeout, interval).Should(And(
			HaveKeyWithValue("username", []byte("admin")),
			HaveKeyWithValue("password", []byte("secret123")),
		))
	})

	// Test 5: Legacy replicate: true should also work
	It("should replicate secrets with legacy replicate: true to new namespace", func() {
		srcNs := createTestNamespace("ns-src-legacy")

		// Create a secret with legacy replicate: true annotation
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "legacy-replicate-secret",
				Namespace: srcNs.Name,
				Annotations: map[string]string{
					ReplicateKey: "true",
				},
			},
			StringData: map[string]string{"key": "legacy"},
			Type:       corev1.SecretTypeOpaque,
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Create a new namespace - the secret should be replicated to it
		targetNs := createTestNamespace("ns-tgt-legacy")

		// Verify the secret was replicated
		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: targetNs.Name}, &corev1.Secret{})
		}, timeout, interval).Should(Succeed())
	})
})

func createTestNamespace(name string) *corev1.Namespace {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns
}

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

var _ = Describe("ConfigMap Replication", func() {
	const (
		timeout  = 30 * time.Second
		interval = 1 * time.Second
	)

	// Test 1: Basic single namespace replication
	It("should replicate configmap to single namespace", func() {
		ns1 := createNamespace("cm-single-src")
		ns2 := createNamespace("cm-single-tgt")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "single-ns-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name,
				},
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 2: Multiple namespace replication
	It("should replicate configmap to multiple namespaces", func() {
		ns1 := createNamespace("cm-multi-src")
		ns2 := createNamespace("cm-multi-tgt1")
		ns3 := createNamespace("cm-multi-tgt2")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multi-ns-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name + "," + ns3.Name,
				},
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns3.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 3: Replicate to all namespaces
	It("should replicate configmap to all namespaces when annotation is true", func() {
		ns1 := createNamespace("cm-all-src")
		ns2 := createNamespace("cm-all-tgt1")
		ns3 := createNamespace("cm-all-tgt2")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "all-ns-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: "true",
				},
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns3.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 4: Skip replication when annotation is empty
	It("should skip replication when annotation is empty", func() {
		ns1 := createNamespace("cm-empty-src")
		ns2 := createNamespace("cm-empty-tgt")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-replicate-cm",
				Namespace: ns1.Name,
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, 5*time.Second, interval).ShouldNot(Succeed())
	})

	// Test 5: Skip replication when annotation is "false"
	It("should skip replication when annotation is false", func() {
		ns1 := createNamespace("cm-false-src")
		ns2 := createNamespace("cm-false-tgt")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "false-replicate-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: "false",
				},
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Consistently(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, 5*time.Second, interval).ShouldNot(Succeed())
	})

	// Test 6: Update existing configmap in target namespace
	It("should update existing configmap in target namespace", func() {
		ns1 := createNamespace("cm-update-src")
		ns2 := createNamespace("cm-update-tgt")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "update-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name,
				},
			},
			Data: map[string]string{"key": "original"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())

		patch := client.MergeFrom(cm.DeepCopy())
		cm.Data = map[string]string{"key": "updated"}
		Expect(k8sClient.Patch(ctx, cm, patch)).To(Succeed())

		Eventually(func() string {
			var replicated corev1.ConfigMap
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &replicated); err != nil {
				return ""
			}
			return replicated.Data["key"]
		}, timeout, interval).Should(Equal("updated"))
	})

	// Test 7: Handle whitespace in namespace list
	It("should handle whitespace in namespace list", func() {
		ns1 := createNamespace("cm-ws-src")
		ns2 := createNamespace("cm-ws-tgt")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "whitespace-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: "  " + ns2.Name + "  ",
				},
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 8: Handle empty entries in namespace list
	It("should handle empty entries in namespace list", func() {
		ns1 := createNamespace("cm-empty-entry-src")
		ns2 := createNamespace("cm-empty-entry-tgt")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "empty-entry-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name + ",,",
				},
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &corev1.ConfigMap{})
		}, timeout, interval).Should(Succeed())
	})

	// Test 9: Exclude source namespace from replication targets
	It("should exclude source namespace from replication targets", func() {
		ns1 := createNamespace("cm-exclude-src")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "exclude-self-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns1.Name,
				},
			},
			Data: map[string]string{"key": "value"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Consistently(func() int {
			var cms corev1.ConfigMapList
			if err := k8sClient.List(ctx, &cms, client.InNamespace(ns1.Name)); err != nil {
				return -1
			}
			count := 0
			for _, c := range cms.Items {
				if c.Name == cm.Name {
					count++
				}
			}
			return count
		}, 5*time.Second, interval).Should(Equal(1))
	})

	// Test 10: Trigger rollout when annotation is true
	It("should trigger rollout when rollout-on-update annotation is true", func() {
		ns := createNamespace("cm-rollout-ns")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rollout-cm",
				Namespace: ns.Name,
				Annotations: map[string]string{
					ReplicateKey:       ns.Name,
					RolloutOnUpdateKey: "true",
				},
			},
			Data: map[string]string{"config": "val"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		deploy := createDeploymentWithConfigMapEnvFrom(ns.Name, "cm-rollout-deploy", cm.Name)
		Expect(k8sClient.Create(ctx, deploy)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &appsv1.Deployment{})
		}, timeout, interval).Should(Succeed())

		patch := client.MergeFrom(cm.DeepCopy())
		cm.Data = map[string]string{"config": "updated"}
		Expect(k8sClient.Patch(ctx, cm, patch)).To(Succeed())

		Eventually(func() string {
			var d appsv1.Deployment
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &d); err != nil {
				return ""
			}
			if d.Spec.Template.Annotations == nil {
				return ""
			}
			return d.Spec.Template.Annotations["configmap.restartedAt"]
		}, timeout, interval).ShouldNot(BeEmpty())
	})

	// Test 11: Replicate data matches source
	It("should replicate configmap data that matches source", func() {
		ns1 := createNamespace("cm-data-src")
		ns2 := createNamespace("cm-data-tgt")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "data-match-cm",
				Namespace: ns1.Name,
				Annotations: map[string]string{
					ReplicateKey: ns2.Name,
				},
			},
			Data: map[string]string{
				"app.conf":  "setting=value",
				"db.config": "host=localhost",
			},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		Eventually(func() map[string]string {
			var replicated corev1.ConfigMap
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: ns2.Name}, &replicated); err != nil {
				return nil
			}
			return replicated.Data
		}, timeout, interval).Should(And(
			HaveKeyWithValue("app.conf", "setting=value"),
			HaveKeyWithValue("db.config", "host=localhost"),
		))
	})

	// Test 12: Deployment with volume mount triggers rollout
	It("should trigger rollout for deployment using configmap as volume", func() {
		ns := createNamespace("cm-vol-rollout-ns")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vol-rollout-cm",
				Namespace: ns.Name,
				Annotations: map[string]string{
					ReplicateKey:       ns.Name,
					RolloutOnUpdateKey: "true",
				},
			},
			Data: map[string]string{"config": "data"},
		}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		deploy := createDeploymentWithConfigMapVolume(ns.Name, "cm-vol-deploy", cm.Name)
		Expect(k8sClient.Create(ctx, deploy)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &appsv1.Deployment{})
		}, timeout, interval).Should(Succeed())

		patch := client.MergeFrom(cm.DeepCopy())
		cm.Data = map[string]string{"config": "updated-data"}
		Expect(k8sClient.Patch(ctx, cm, patch)).To(Succeed())

		Eventually(func() string {
			var d appsv1.Deployment
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: ns.Name}, &d); err != nil {
				return ""
			}
			if d.Spec.Template.Annotations == nil {
				return ""
			}
			return d.Spec.Template.Annotations["configmap.restartedAt"]
		}, timeout, interval).ShouldNot(BeEmpty())
	})
})

// Helper functions for ConfigMap tests

func createDeploymentWithConfigMapEnvFrom(namespace, name, cmName string) *appsv1.Deployment {
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
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
							},
						}},
					}},
				},
			},
		},
	}
}

func createDeploymentWithConfigMapVolume(namespace, name, cmName string) *appsv1.Deployment {
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
							Name:      "config-vol",
							MountPath: "/config",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-vol",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
							},
						},
					}},
				},
			},
		},
	}
}

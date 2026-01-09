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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestParseReplicationConfig_EmptyAnnotations(t *testing.T) {
	config := ParseReplicationConfig(nil, "source-ns")
	if !config.SkipReplication {
		t.Error("expected SkipReplication to be true for empty annotations")
	}
	if config.RolloutOnUpdate {
		t.Error("expected RolloutOnUpdate to be false")
	}
	if config.ReplicateAll {
		t.Error("expected ReplicateAll to be false")
	}
}

func TestParseReplicationConfig_False(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey: "false",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if !config.SkipReplication {
		t.Error("expected SkipReplication to be true for 'false' annotation")
	}
}

func TestParseReplicationConfig_True(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey: "true",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if config.SkipReplication {
		t.Error("expected SkipReplication to be false")
	}
	if !config.ReplicateAll {
		t.Error("expected ReplicateAll to be true")
	}
}

func TestParseReplicationConfig_SingleNamespace(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey: "target-ns",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if config.SkipReplication {
		t.Error("expected SkipReplication to be false")
	}
	if config.ReplicateAll {
		t.Error("expected ReplicateAll to be false")
	}
	if len(config.TargetNamespaces) != 1 || config.TargetNamespaces[0] != "target-ns" {
		t.Errorf("expected TargetNamespaces to be ['target-ns'], got %v", config.TargetNamespaces)
	}
}

func TestParseReplicationConfig_MultipleNamespaces(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey: "ns1,ns2,ns3",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if len(config.TargetNamespaces) != 3 {
		t.Errorf("expected 3 target namespaces, got %d", len(config.TargetNamespaces))
	}
}

func TestParseReplicationConfig_TrimWhitespace(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey: "  ns1  ,  ns2  ",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if len(config.TargetNamespaces) != 2 {
		t.Errorf("expected 2 target namespaces, got %d", len(config.TargetNamespaces))
	}
	for _, ns := range config.TargetNamespaces {
		if ns != "ns1" && ns != "ns2" {
			t.Errorf("unexpected namespace: %s", ns)
		}
	}
}

func TestParseReplicationConfig_FilterEmptyEntries(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey: "ns1,,ns2,",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if len(config.TargetNamespaces) != 2 {
		t.Errorf("expected 2 target namespaces, got %d", len(config.TargetNamespaces))
	}
}

func TestParseReplicationConfig_ExcludeSourceNamespace(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey: "source-ns,target-ns",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if len(config.TargetNamespaces) != 1 || config.TargetNamespaces[0] != "target-ns" {
		t.Errorf("expected source namespace to be excluded, got %v", config.TargetNamespaces)
	}
}

func TestParseReplicationConfig_RolloutOnUpdate(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey:       "target-ns",
		RolloutOnUpdateKey: "true",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if !config.RolloutOnUpdate {
		t.Error("expected RolloutOnUpdate to be true")
	}
}

func TestParseReplicationConfig_RolloutOnUpdateFalse(t *testing.T) {
	annotations := map[string]string{
		ReplicateKey:       "target-ns",
		RolloutOnUpdateKey: "false",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if config.RolloutOnUpdate {
		t.Error("expected RolloutOnUpdate to be false")
	}
}

func TestParseReplicationConfig_ReplicateAll(t *testing.T) {
	annotations := map[string]string{
		ReplicateAllKey: "true",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if config.SkipReplication {
		t.Error("expected SkipReplication to be false")
	}
	if !config.ReplicateAll {
		t.Error("expected ReplicateAll to be true")
	}
}

func TestParseReplicationConfig_ReplicateAllWithNamespaces(t *testing.T) {
	// replicate-all takes precedence over replicate namespace list
	annotations := map[string]string{
		ReplicateAllKey: "true",
		ReplicateKey:    "ns1,ns2",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if !config.ReplicateAll {
		t.Error("expected ReplicateAll to be true when replicate-all is set")
	}
	if len(config.TargetNamespaces) != 0 {
		t.Errorf("expected empty TargetNamespaces when replicate-all is true, got %v", config.TargetNamespaces)
	}
}

func TestParseReplicationConfig_ReplicateAllFalse(t *testing.T) {
	// replicate-all: "false" should skip replication if replicate is not set
	annotations := map[string]string{
		ReplicateAllKey: "false",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if !config.SkipReplication {
		t.Error("expected SkipReplication to be true when replicate-all is false and replicate is not set")
	}
}

func TestParseReplicationConfig_ReplicateAllFalseWithNamespaces(t *testing.T) {
	// replicate-all: "false" with replicate set should use the namespace list
	annotations := map[string]string{
		ReplicateAllKey: "false",
		ReplicateKey:    "ns1,ns2",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if config.SkipReplication {
		t.Error("expected SkipReplication to be false when replicate has namespaces")
	}
	if config.ReplicateAll {
		t.Error("expected ReplicateAll to be false")
	}
	if len(config.TargetNamespaces) != 2 {
		t.Errorf("expected 2 target namespaces, got %d", len(config.TargetNamespaces))
	}
}

func TestParseReplicationConfig_NamespaceNamedTrue(t *testing.T) {
	// A namespace literally named "true" should work when using replicate annotation
	annotations := map[string]string{
		ReplicateKey: "true", // Legacy: this means all namespaces
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	if !config.ReplicateAll {
		t.Error("expected ReplicateAll to be true for legacy 'true' value")
	}
}

func TestParseReplicationConfig_NamespaceNamedTrueWithReplicateAll(t *testing.T) {
	// To target a namespace literally named "true", use replicate-all: "false" and replicate: "true"
	annotations := map[string]string{
		ReplicateAllKey: "false",
		ReplicateKey:    "true",
	}
	config := ParseReplicationConfig(annotations, "source-ns")
	// With replicate-all explicitly false, replicate: "true" is treated as namespace name
	if config.ReplicateAll {
		t.Error("expected ReplicateAll to be false when replicate-all is explicitly false")
	}
	if config.SkipReplication {
		t.Error("expected SkipReplication to be false")
	}
	if len(config.TargetNamespaces) != 1 || config.TargetNamespaces[0] != "true" {
		t.Errorf("expected TargetNamespaces to be ['true'], got %v", config.TargetNamespaces)
	}
}

func TestIsDeploymentUsingSecret_EnvFrom(t *testing.T) {
	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						EnvFrom: []corev1.EnvFromSource{{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
							},
						}},
					}},
				},
			},
		},
	}
	if !IsDeploymentUsingSecret(deploy, "my-secret") {
		t.Error("expected deployment to be using my-secret")
	}
	if IsDeploymentUsingSecret(deploy, "other-secret") {
		t.Error("expected deployment to not be using other-secret")
	}
}

func TestIsDeploymentUsingSecret_Volume(t *testing.T) {
	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "my-secret",
							},
						},
					}},
				},
			},
		},
	}
	if !IsDeploymentUsingSecret(deploy, "my-secret") {
		t.Error("expected deployment to be using my-secret")
	}
	if IsDeploymentUsingSecret(deploy, "other-secret") {
		t.Error("expected deployment to not be using other-secret")
	}
}

func TestIsDeploymentUsingSecret_NotUsed(t *testing.T) {
	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "main",
						Image: "busybox",
					}},
				},
			},
		},
	}
	if IsDeploymentUsingSecret(deploy, "my-secret") {
		t.Error("expected deployment to not be using any secret")
	}
}

func TestIsDeploymentUsingConfigMap_EnvFrom(t *testing.T) {
	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						EnvFrom: []corev1.EnvFromSource{{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "my-configmap"},
							},
						}},
					}},
				},
			},
		},
	}
	if !IsDeploymentUsingConfigMap(deploy, "my-configmap") {
		t.Error("expected deployment to be using my-configmap")
	}
	if IsDeploymentUsingConfigMap(deploy, "other-configmap") {
		t.Error("expected deployment to not be using other-configmap")
	}
}

func TestIsDeploymentUsingConfigMap_Volume(t *testing.T) {
	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "my-configmap"},
							},
						},
					}},
				},
			},
		},
	}
	if !IsDeploymentUsingConfigMap(deploy, "my-configmap") {
		t.Error("expected deployment to be using my-configmap")
	}
	if IsDeploymentUsingConfigMap(deploy, "other-configmap") {
		t.Error("expected deployment to not be using other-configmap")
	}
}

func TestIsDeploymentUsingConfigMap_NotUsed(t *testing.T) {
	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "main",
						Image: "busybox",
					}},
				},
			},
		},
	}
	if IsDeploymentUsingConfigMap(deploy, "my-configmap") {
		t.Error("expected deployment to not be using any configmap")
	}
}

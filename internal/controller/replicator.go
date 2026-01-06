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
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Shared annotation keys for replication configuration
const (
	ReplicateKey       = "replizieren.dev/replicate"
	RolloutOnUpdateKey = "replizieren.dev/rollout-on-update"
)

// ReplicationConfig holds parsed annotation configuration
type ReplicationConfig struct {
	TargetNamespaces []string
	ReplicateAll     bool
	RolloutOnUpdate  bool
	SkipReplication  bool
}

// ParseReplicationConfig extracts replication settings from annotations
func ParseReplicationConfig(annotations map[string]string, sourceNamespace string) ReplicationConfig {
	replicateTo := annotations[ReplicateKey]
	rollout := annotations[RolloutOnUpdateKey] == "true"

	config := ReplicationConfig{
		RolloutOnUpdate: rollout,
	}

	if replicateTo == "" || replicateTo == "false" {
		config.SkipReplication = true
		return config
	}

	if replicateTo == "true" {
		config.ReplicateAll = true
		return config
	}

	// Parse comma-separated namespace list
	for _, ns := range strings.Split(replicateTo, ",") {
		ns = strings.TrimSpace(ns)
		if ns != "" && ns != sourceNamespace {
			config.TargetNamespaces = append(config.TargetNamespaces, ns)
		}
	}

	return config
}

// GetAllNamespaces returns all namespace names except the excluded one
func GetAllNamespaces(ctx context.Context, c client.Client, excludeNamespace string) ([]string, error) {
	var nsList corev1.NamespaceList
	if err := c.List(ctx, &nsList); err != nil {
		return nil, err
	}

	var namespaces []string
	for _, ns := range nsList.Items {
		if ns.Name != excludeNamespace {
			namespaces = append(namespaces, ns.Name)
		}
	}
	return namespaces, nil
}

// RestartDeploymentsFunc is a function type that checks if a deployment uses a resource
type RestartDeploymentsFunc func(*appsv1.Deployment) bool

// RestartDeployments patches deployments that use the specified resource
func RestartDeployments(
	ctx context.Context,
	c client.Client,
	namespace string,
	annotationKey string,
	isUsing RestartDeploymentsFunc,
) error {
	var deploys appsv1.DeploymentList
	if err := c.List(ctx, &deploys, client.InNamespace(namespace)); err != nil {
		return err
	}

	for _, deploy := range deploys.Items {
		if isUsing(&deploy) {
			patch := client.MergeFrom(deploy.DeepCopy())
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = map[string]string{}
			}
			deploy.Spec.Template.Annotations[annotationKey] = time.Now().Format(time.RFC3339)
			if err := c.Patch(ctx, &deploy, patch); err != nil {
				return fmt.Errorf("failed to patch deployment %s: %w", deploy.Name, err)
			}
		}
	}
	return nil
}

// IsDeploymentUsingSecret checks if a deployment uses the named secret
func IsDeploymentUsingSecret(deploy *appsv1.Deployment, secretName string) bool {
	for _, vol := range deploy.Spec.Template.Spec.Volumes {
		if vol.Secret != nil && vol.Secret.SecretName == secretName {
			return true
		}
	}
	for _, c := range deploy.Spec.Template.Spec.Containers {
		for _, envFrom := range c.EnvFrom {
			if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secretName {
				return true
			}
		}
	}
	return false
}

// IsDeploymentUsingConfigMap checks if a deployment uses the named configmap
func IsDeploymentUsingConfigMap(deploy *appsv1.Deployment, cmName string) bool {
	for _, vol := range deploy.Spec.Template.Spec.Volumes {
		if vol.ConfigMap != nil && vol.ConfigMap.Name == cmName {
			return true
		}
	}
	for _, c := range deploy.Spec.Template.Spec.Containers {
		for _, envFrom := range c.EnvFrom {
			if envFrom.ConfigMapRef != nil && envFrom.ConfigMapRef.Name == cmName {
				return true
			}
		}
	}
	return false
}

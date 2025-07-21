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
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ConfigMapWatcherReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	replicateKeyCM       = "replizieren.dev/replicate"
	rolloutOnUpdateKeyCM = "replizieren.dev/rollout-on-update"
)

func (r *ConfigMapWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cm corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &cm); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	const replicateTo = cm.Annotations[replicateKeyCM]
	const rollout = cm.Annotations[rolloutOnUpdateKeyCM] == "true"

	if replicateTo == "" || replicateTo == "false" && !rollout {
		logger.Info("Replication not set, skipping")
		return ctrl.Result{}, nil
	}

	var targetNamespaces []string
	if replicateTo == "true" {
		var nsList corev1.NamespaceList
		if err := r.List(ctx, &nsList); err != nil {
			return ctrl.Result{}, err
		}
		for _, ns := range nsList.Items {
			if ns.Name != cm.Namespace {
				targetNamespaces = append(targetNamespaces, ns.Name)
			}
		}
	} else {
		targetNamespaces = strings.Split(replicateTo, ",")
	}

	for _, ns := range targetNamespaces {
		if replicateTo != "false" && replicateTo != "" {
			if err := r.replicateConfigMap(ctx, &cm, ns); err != nil {
				logger.Error(err, "Failed to replicate configmap", "namespace", ns)
				continue
			}
		}
		if rollout {
			_ = r.restartDeploymentsUsingConfigMap(ctx, cm.Name, ns)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ConfigMapWatcherReconciler) replicateConfigMap(ctx context.Context, original *corev1.ConfigMap, namespace string) error {
	clone := original.DeepCopy()
	clone.Namespace = namespace
	clone.ResourceVersion = ""
	clone.UID = ""

	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: clone.Name, Namespace: namespace}, existing)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, clone)
	} else if err != nil {
		return err
	}

	clone.ResourceVersion = existing.ResourceVersion
	return r.Update(ctx, clone)
}

func (r *ConfigMapWatcherReconciler) restartDeploymentsUsingConfigMap(ctx context.Context, cmName, namespace string) error {
	var deploys appsv1.DeploymentList
	if err := r.List(ctx, &deploys, client.InNamespace(namespace)); err != nil {
		return err
	}

	for _, deploy := range deploys.Items {
		if isUsingConfigMap(&deploy, cmName) {
			patch := client.MergeFrom(deploy.DeepCopy())
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = map[string]string{}
			}
			deploy.Spec.Template.Annotations["configmap.restartedAt"] = time.Now().Format(time.RFC3339)
			_ = r.Patch(ctx, &deploy, patch)
		}
	}
	return nil
}

func isUsingConfigMap(deploy *appsv1.Deployment, cmName string) bool {
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

func (r *ConfigMapWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("configmapwatcher").
		Complete(r)
}

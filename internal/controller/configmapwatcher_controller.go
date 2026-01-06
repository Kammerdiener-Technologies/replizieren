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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ConfigMapWatcherReconciler reconciles a ConfigMap object
type ConfigMapWatcherReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;patch

// Reconcile handles ConfigMap replication and deployment rollout triggers.
func (r *ConfigMapWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cm corev1.ConfigMap
	if err := r.Get(ctx, req.NamespacedName, &cm); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	config := ParseReplicationConfig(cm.Annotations, cm.Namespace)

	if config.SkipReplication && !config.RolloutOnUpdate {
		logger.Info("Replication not set, skipping")
		return ctrl.Result{}, nil
	}

	targetNamespaces := config.TargetNamespaces
	if config.ReplicateAll {
		var err error
		targetNamespaces, err = GetAllNamespaces(ctx, r.Client, cm.Namespace)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, ns := range targetNamespaces {
		if !config.SkipReplication {
			if err := r.replicateConfigMap(ctx, &cm, ns); err != nil {
				logger.Error(err, "Failed to replicate configmap", "namespace", ns)
				continue
			}
		}
		if config.RolloutOnUpdate {
			if err := RestartDeployments(ctx, r.Client, ns, "configmap.restartedAt", func(d *appsv1.Deployment) bool {
				return IsDeploymentUsingConfigMap(d, cm.Name)
			}); err != nil {
				logger.Error(err, "Failed to restart deployments", "namespace", ns)
			}
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

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Named("configmapwatcher").
		Complete(r)
}

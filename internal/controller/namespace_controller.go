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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NamespaceReconciler reconciles a Namespace object to trigger replication
// of secrets and configmaps that have replicate-all enabled
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

// Reconcile handles namespace creation events and replicates secrets/configmaps
// that have replicate-all annotation to the new namespace.
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var namespace corev1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &namespace); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Skip if namespace is being deleted
	if namespace.DeletionTimestamp != nil {
		logger.Info("Namespace is being deleted, skipping", "namespace", namespace.Name)
		return ctrl.Result{}, nil
	}

	// Skip system namespaces
	if IsSystemNamespace(namespace.Name) {
		logger.Info("Skipping system namespace", "namespace", namespace.Name)
		return ctrl.Result{}, nil
	}

	logger.Info("New namespace detected, checking for resources to replicate", "namespace", namespace.Name)

	// Replicate secrets with replicate-all annotation
	secrets, err := GetSecretsToReplicateAll(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to list secrets for replication")
		return ctrl.Result{}, err
	}

	for _, secret := range secrets {
		if secret.Namespace == namespace.Name {
			continue // Don't replicate to source namespace
		}
		if err := r.replicateSecret(ctx, &secret, namespace.Name); err != nil {
			logger.Error(err, "Failed to replicate secret", "secret", secret.Name, "from", secret.Namespace, "to", namespace.Name)
			continue
		}
		logger.Info("Replicated secret to new namespace", "secret", secret.Name, "from", secret.Namespace, "to", namespace.Name)
	}

	// Replicate configmaps with replicate-all annotation
	configmaps, err := GetConfigMapsToReplicateAll(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to list configmaps for replication")
		return ctrl.Result{}, err
	}

	for _, cm := range configmaps {
		if cm.Namespace == namespace.Name {
			continue // Don't replicate to source namespace
		}
		if err := r.replicateConfigMap(ctx, &cm, namespace.Name); err != nil {
			logger.Error(err, "Failed to replicate configmap", "configmap", cm.Name, "from", cm.Namespace, "to", namespace.Name)
			continue
		}
		logger.Info("Replicated configmap to new namespace", "configmap", cm.Name, "from", cm.Namespace, "to", namespace.Name)
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) replicateSecret(ctx context.Context, original *corev1.Secret, namespace string) error {
	clone := original.DeepCopy()
	clone.Namespace = namespace
	clone.ResourceVersion = ""
	clone.UID = ""

	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: clone.Name, Namespace: namespace}, existing)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, clone)
	} else if err != nil {
		return err
	}

	clone.ResourceVersion = existing.ResourceVersion
	return r.Update(ctx, clone)
}

func (r *NamespaceReconciler) replicateConfigMap(ctx context.Context, original *corev1.ConfigMap, namespace string) error {
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
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("namespace").
		Complete(r)
}

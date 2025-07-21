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

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	replicateKeyS       = "replizieren.dev/replicate"
	rolloutOnUpdateKeyS = "replizieren.dev/rollout-on-update"
)

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Secret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	replicateTo := secret.Annotations[replicateKeyS]
	rollout := secret.Annotations[rolloutOnUpdateKeyS] == "true"

	if replicateTo == "" || replicateTo == "false" && rollout == false {
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
			if ns.Name != secret.Namespace {
				targetNamespaces = append(targetNamespaces, ns.Name)
			}
		}
	} else {
		targetNamespaces = strings.Split(replicateTo, ",")
	}

	for _, ns := range targetNamespaces {
		if replicateTo != "false" && replicateTo != "" {
			if err := r.replicateSecret(ctx, &secret, ns); err != nil {
				logger.Error(err, "Failed to replicate secret", "namespace", ns)
				continue
			}
		}
		if rollout {
			_ = r.restartDeploymentsUsingSecret(ctx, secret.Name, ns)
		}
	}

	return ctrl.Result{}, nil
}

func (r *SecretReconciler) replicateSecret(ctx context.Context, original *corev1.Secret, namespace string) error {
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

func (r *SecretReconciler) restartDeploymentsUsingSecret(ctx context.Context, secretName, namespace string) error {
	var deploys appsv1.DeploymentList
	if err := r.List(ctx, &deploys, client.InNamespace(namespace)); err != nil {
		return err
	}

	for _, deploy := range deploys.Items {
		if isUsingSecret(&deploy, secretName) {
			patch := client.MergeFrom(deploy.DeepCopy())
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = map[string]string{}
			}
			deploy.Spec.Template.Annotations["secret.restartedAt"] = time.Now().Format(time.RFC3339)
			_ = r.Patch(ctx, &deploy, patch)
		}
	}
	return nil
}

func isUsingSecret(deploy *appsv1.Deployment, secretName string) bool {
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

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Named("secret").
		Complete(r)
}

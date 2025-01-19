/*
Copyright 2025 drumato.

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
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SecretReconciler reconciles a Secret object
type SecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	TTLAfterAgeAnnotation = "ttlsecret.drumato.com/ttl-after-age"
	TTLAnnotation         = "ttlsecret.drumato.com/ttl"
)

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(9).Info("start reconciliation")

	secret := corev1.Secret{}
	if err := r.Client.Get(ctx, req.NamespacedName, &secret); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	var ttl time.Time
	ttlAnnotation := secret.GetAnnotations()[TTLAnnotation]
	if ttlAnnotation == "" {
		ttlAnnotation = secret.GetAnnotations()[TTLAfterAgeAnnotation]
		if ttlAnnotation == "" {
			return ctrl.Result{}, nil
		}

		t, err := time.Parse(time.RFC3339, ttlAnnotation)
		if err != nil {
			return ctrl.Result{}, err
		}
		ttl = t
	}

	if time.Now().Before(ttl) {
		logger.V(0).Info("will enqueue after ttl will be expired")
		return ctrl.Result{RequeueAfter: time.Until(ttl)}, nil
	}

	logger.V(0).Info("delete secret", "namespace", secret.GetNamespace(), "name", secret.GetName())
	if err := r.Client.Delete(ctx, &secret); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.AnnotationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(p)).
		Named("secret").
		Complete(r)
}

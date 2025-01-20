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
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	TTLLayoutAnnotation   = "ttlsecret.drumato.com/ttl-layout"
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
	if err := r.Client.Get(ctx, req.NamespacedName, &secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	annotations := secret.GetAnnotations()

	var ttl time.Time
	switch {
	case annotations[TTLAfterAgeAnnotation] != "":
		d, err := parseDuration(annotations[TTLAfterAgeAnnotation])
		if err != nil {
			return ctrl.Result{}, err
		}
		ttl = secret.GetCreationTimestamp().UTC().Add(d)
	case annotations[TTLAnnotation] != "":
		layout := time.RFC3339
		if l := annotations[TTLLayoutAnnotation]; l != "" {
			layout = convertLayout(l)
		}
		t, err := time.Parse(layout, annotations[TTLAnnotation])
		if err != nil {
			return ctrl.Result{}, err
		}
		ttl = t.UTC()
		logger.V(0).Info(ttl.String())
	default:
		return ctrl.Result{}, fmt.Errorf("'ttlsecret.drumato.com/ttl' or 'ttlsecret.drumato.com/ttl-after-age' must be specified")
	}

	now := time.Now().UTC()
	if now.Before(ttl) {
		logger.V(0).Info("will enqueue after ttl will be expired")
		return ctrl.Result{RequeueAfter: ttl.Sub(now)}, nil
	}

	logger.V(0).Info("delete secret", "namespace", secret.GetNamespace(), "name", secret.GetName())
	if err := r.Client.Delete(ctx, &secret); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func convertLayout(l string) string {
	switch strings.ToLower(l) {
	case "rfc822":
		return time.RFC822
	case "rfc822z":
		return time.RFC822Z
	case "rfc1123":
		return time.RFC1123
	case "rfc1123z":
		return time.RFC1123Z
	case "kitchen":
		return time.Kitchen
	case "rfc3339":
		return time.RFC3339
	case "rfc3339nano":
		return time.RFC3339Nano
	case "datetime":
		return time.DateTime
	case "dateonly":
		return time.DateOnly
	case "timeonly":
		return time.TimeOnly
	default:
		return l
	}
}

var reDuration = regexp.MustCompile(`^(\d+)([dhms])$`)

// parseDuration parses the duration representation like (3d/2h/15m/10s).
func parseDuration(s string) (time.Duration, error) {
	match := reDuration.FindStringSubmatch(s)
	if match == nil {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}

	switch match[2] {
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "s":
		return time.Duration(value) * time.Second, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", match[2])
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.AnnotationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(p)).
		Named("secret").
		Complete(r)
}

/*
Copyright 2022.

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

package openfeature

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	featurev1 "github.com/open-feature/feature-operator/pkg/apis/open-feature.dev/v1alpha1"
	"github.com/open-feature/feature-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OpenFeatureReconciler reconciles a OpenFeature object
type OpenFeatureReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Log      logr.Logger
}

//+kubebuilder:rbac:groups=cache.openfeature.dev,resources=openfeatures,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.openfeature.dev,resources=openfeatures/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.openfeature.dev,resources=openfeatures/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenFeature object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OpenFeatureReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = common.InitLog(req, "OpenFeature Controller", common.CONTROLLER_OPENFEATURE_NAME)

	instance := &featurev1.OpenFeature{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info(fmt.Sprintf("%s resource not found. Ignoring since object must be deleted", common.CRD_OPENFEATURE_NAME))
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "Failed to get "+common.CRD_OPENFEATURE_NAME)
		return ctrl.Result{}, err
	}

	podList, err := r.podList(ctx, instance)
	if err != nil {
		r.Log.Error(err, "Could not list pods")
	}

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podList, instance.Status.Pods) {
		instance.Status.Pods = podList
		err := r.Status().Update(ctx, instance)
		if err != nil {
			r.Log.Error(err, fmt.Sprintf("Failed to update %s status", common.CRD_OPENFEATURE_NAME))
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenFeatureReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&featurev1.OpenFeature{}).
		Complete(r)
}

func (r *OpenFeatureReconciler) podList(ctx context.Context, feature *featurev1.OpenFeature) ([]string, error) {
	var podNames []string
	labels := map[string]string{"app": common.FLAG_API_DEPLOYMENT_NAME, "openfeature_cr": feature.Name}

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(feature.Namespace),
		client.MatchingLabels(labels),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		r.Log.Error(err, "Failed to list pods", common.CRD_OPENFEATURE_NAME+".Namespace", feature.Namespace, common.CRD_OPENFEATURE_NAME+".Name", feature.Name)
		return nil, err
	}

	for _, pod := range podList.Items {
		podNames = append(podNames, pod.Name)
	}

	return podNames, nil
}

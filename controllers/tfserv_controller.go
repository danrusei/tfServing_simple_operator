/*
Copyright 2019 Dan Rusei.

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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servapiv1alpha1 "github.com/Danr17/tfServing_simple_operator/api/v1alpha1"
)

// TfservReconciler reconciles a Tfserv object
type TfservReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=servapi.dev-state.com,resources=tfservs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=servapi.dev-state.com,resources=tfservs/status,verbs=get;update;patch

func (r *TfservReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("tfserv", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *TfservReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servapiv1alpha1.Tfserv{}).
		Complete(r)
}

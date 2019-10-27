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
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	servapiv1alpha1 "github.com/Danr17/tfServing_simple_operator/api/v1alpha1"
)

// TfservReconciler reconciles a Tfserv object
type TfservReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	//	recorder record.EventRecorder
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

// +kubebuilder:rbac:groups=servapi.dev-state.com,resources=tfservs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=servapi.dev-state.com,resources=tfservs/status,verbs=get;update;patch

//Reconcile method
func (r *TfservReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("tfserv", req.NamespacedName)

	var tfs servapiv1alpha1.Tfserv
	if err := r.Get(ctx, req.NamespacedName, &tfs); err != nil {
		log.Error(err, "unable to fetch tfs")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}

	// Verify that ConfigMap exist, do not continue until we find it
	var configmap *corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: tfs.Spec.ConfigMap, Namespace: tfs.Namespace}, configmap)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("ConfigMap %s not found, wont continue untill config map is found\n", tfs.Spec.ConfigMap)
		return ctrl.Result{}, err
	}

	//Set instance as the owner and controller for this configmap
	if err := controllerutil.SetControllerReference(&tfs, configmap, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	//currentConfigVersion := configmap.ResourceVersion
	//TODO Store the configversion annotation deployment or tfs Status
	//TODO If Deployment is already running, check if the configmap version changed. If it does, delete and redeploy.

	labels := map[string]string{
		"tfsName": tfs.Name,
	}

	// Define the desired Deployment object
	var deployment *appsv1.Deployment
	deployment, err = r.createDeployment(&tfs, labels)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if the Deployment already exists
	foundDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	//If the deployment does not exist, create it
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating Deployment %s/%s\n", deployment.Namespace, deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			return ctrl.Result{}, err
		}
		//	r.recorder.Eventf(tfs, "Normal", "DeploymentCreated", "The Deployment %s has been created", deployment.Name)
		log.V(1).Info("The Deployment %s has been created\n", deployment.Name)
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Update the found deployment and write the result back if there are any changes
	if !reflect.DeepEqual(deployment.Spec, foundDeployment.Spec) {
		foundDeployment.Spec = deployment.Spec
		log.V(1).Info("Updating Deployment %s/%s\n", deployment.Namespace, deployment.Name)
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	//Define the desired Service object

	var service *corev1.Service
	service, err = r.createService(&tfs, labels)
	if err != nil {
		return ctrl.Result{}, err
	}

	//check if the service already exists
	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	//If the service does not exist, create it
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating a new Service", "Service.Namespace", service.Name, "Service.Name", service.Name)
		err = r.Create(ctx, service)
		if err != nil {
			return reconcile.Result{}, err
		}
		log.V(1).Info("The Deployment %s has been created\n", deployment.Name)
		return ctrl.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// your logic here

	return ctrl.Result{}, nil
}

func (r *TfservReconciler) createDeployment(tfs *servapiv1alpha1.Tfserv, labels map[string]string) (*appsv1.Deployment, error) {

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tfs.Name,
			Namespace: tfs.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &tfs.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   tfs.Name,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "tfs_main",
							Image:   "tensorflow/serving:latest",
							Command: []string{"/usr/bin/tensorflow_model_server"},
							Args: []string{
								fmt.Sprintf("--port=%d", tfs.Spec.GrpcPort),
								fmt.Sprintf("--rest_api_port=%d", tfs.Spec.RestPort),
								fmt.Sprintf("--model_config_file=%s%s", tfs.Spec.ConfigFileLocation, tfs.Spec.ConfigFileName),
							},
							Env: []corev1.EnvVar{
								{
									Name:  "GOOGLE_APPLICATION_CREDENTIALS",
									Value: tfs.Spec.SecretFileLocation + tfs.Spec.SecretFileName,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: tfs.Spec.GrpcPort,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: tfs.Spec.RestPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tfs-config-volume",
									MountPath: tfs.Spec.ConfigFileLocation,
								},
								{
									Name:      "tfs-secret-volume",
									MountPath: tfs.Spec.SecretFileLocation,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tfs-config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "tfs-config",
									},
								},
							},
						},
						{
							Name: "tfs-secret-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "tfs-secret",
									//TODO I may need to add
									//items:
									// - key: service-account-key
									//   path: key.json
								},
							},
						},
					},
				},
			},
		},
	}

	// SetControllerReference sets owner as a Controller OwnerReference on owned.
	// This is used for garbage collection of the owned object and for
	// reconciling the owner object on changes to owned (with a Watch + EnqueueRequestForOwner).

	if err := controllerutil.SetControllerReference(tfs, deployment, r.Scheme); err != nil {
		return nil, err
	}

	return deployment, nil
}

func (r *TfservReconciler) createService(tfs *servapiv1alpha1.Tfserv, labels map[string]string) (*corev1.Service, error) {

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tfs.Name + "-service",
			Namespace: tfs.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "rest",
					Port:     tfs.Spec.RestPort,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "grpcs",
					Port:     tfs.Spec.GrpcPort,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{"app": tfs.Name},
		},
	}

	// SetControllerReference sets owner as a Controller OwnerReference on owned.
	// This is used for garbage collection of the owned object and for
	// reconciling the owner object on changes to owned (with a Watch + EnqueueRequestForOwner).

	if err := controllerutil.SetControllerReference(tfs, service, r.Scheme); err != nil {
		return nil, err
	}

	return service, nil
}

//SetupWithManager setup the controler with manager
func (r *TfservReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servapiv1alpha1.Tfserv{}).
		Complete(r)
}

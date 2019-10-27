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
	"log"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	tfs := &servapiv1alpha1.Tfserv{}

	labels := map[string]string{
		"tfsName": tfs.Name,
	}

	var deployment *appsv1.Deployment
	// Got the Website resource instance, now reconcile owned Deployment and Service resources
	deployment, err := r.createDeployment(tfs, labels)
	if err != nil {
		return ctrl.Result{}, err
	}

	var service *corev1.Service
	// Now reconcile the Service that is owned by the Website resource
	service, err = r.createService(tfs, labels)
	if err != nil {
		return ctrl.Result{}, err
	}

	var configmap *corev1.ConfigMap
	// Now reconcile the Service that is owned by the Website resource
	configmap, err = r.createConfigMap(tfs, labels)
	if err != nil {
		return ctrl.Result{}, err
	}

	//TODO ! temp to stop errors, but we have to remove below lines
	log.Printf("this is deployment: %v", deployment)
	log.Printf("this is service: %v", service)
	log.Printf("this is configmap: %v", configmap)

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
								fmt.Sprintf("--model_config_file=%s%s", tfs.Spec.ModelConfigLocation, tfs.Spec.ModelConfigFile),
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
									MountPath: tfs.Spec.ModelConfigLocation,
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
					},
				},
			},
		},
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

	return service, nil
}

func (r *TfservReconciler) createConfigMap(tfs *servapiv1alpha1.Tfserv, labels map[string]string) (*corev1.ConfigMap, error) {

	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: tfs.Name + "-config",
		},
		/*Data: map[string]string{
			tfs.Spec.ModelConfigFile: +
			model_config_list: {
				config: {
				  name: "resnet",
				  base_path: "s3://ml-models-repository/resnet",
				  model_platform: "tensorflow",
				  model_version_policy: {
					specific: {
					  versions: 1
					}
				  }
				}
			  },
		},
		*/

	}
	return configmap, nil
}

func (r *TfservReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&servapiv1alpha1.Tfserv{}).
		Complete(r)
}

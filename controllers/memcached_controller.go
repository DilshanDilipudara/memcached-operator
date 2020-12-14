/*


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
	"reflect"

	cachev1alpha1 "github.com/DilshanDilipudara/memcached-operator.git/api/v1alpha1"
	"github.com/docker/docker/image/cache"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MemcachedReconciler reconciles a Memcached object
type MemcachedReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch

func (r *MemcachedReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("memcached", req.NamespacedName)

	//Fetch the app instance
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(context.TODO(),req.NamespacedName,memcached)
	if err != nil {
		if errors.IsNotFound(err){
			//Request object not found, could have been deleted after reconcile request.
			//Owned objects are automatically garbage collected. For additional cleanup logic use finalizers
			// Return and don't requeue
			return ctrl.Result{},nil
		}
		return ctrl.Result{},err
	}

	// Check if the deployment already exists, if not create a new deployment.
	found := &appsv1.Deployment{}
	err = r.get(context.TODO(),type.NamespacedName{Name: memcached.Name,Namespace:memcached.Namespace},found)
	if err != nil{
		if errors.IsNotFound(err){
			dep := r.deploymentForApp(memcached)
			if err = r.Create(context.TODO(),dep); err != nil{
				return ctrl.Result{},err
			}
			return ctrl.Result{Requeue:true},nil
		} else {
			return ctrl.Result{},err
		}
	}

	//Ensure Deployemt size is same as the spec
	size := memcached.Spec.Size
	if *found.Spec.Replicas != size{
		found.Spec.Replicas = size
		if err = r.Update(context.TODO(),found); err != nil{
			return ctrl.Result{},err
		}
		return ctrl.Result{Requeue:true},nil
	}

	//Update the Memcahed status with the pod name
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(app.Namespace),
		client.MatchingLabels(labelsForApp(memcached.Name))
	}
	if err = r.List(context.TODO(),podList,listOpts...); err != nil{
		return ctrl.Result{},err
	}

	// Update status.Nodes if needed.
	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames,memcached.Status.Nodes){
		memcached.Status.Nodes = podNames
		if err := r.Status().Update(context.TODO(),mecached); err != nil{
			return ctrl.Result{},err
		}
	}


	return ctrl.Result{}, nil
}


// deploymentForApp returns a app Deployment object.
func (r *KindReconciler) deploymentForApp(m *cachev1alpha1.Memcached) *appsv1.Deployment {
    lbls := labelsForApp(m.Name)
    replicas := m.Spec.Size

    dep := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      m.Name,
            Namespace: m.Namespace,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: lbls,
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: lbls,
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Image:   "app:alpine",
                        Name:    "app",
        		        Command: []string{"app", "-a=64", "-b"},
                        Ports: []corev1.ContainerPort{{
                            ContainerPort: 10000,
                            Name:          "app",
                        }},
                    }},
                },
            },
        },
    }

    // Set App instance as the owner and controller.
    // NOTE: calling SetControllerReference, and setting owner references in
    // general, is important as it allows deleted objects to be garbage collected.
    controllerutil.SetControllerReference(m, dep, r.scheme)
    return dep
}

// labelsForApp creates a simple set of labels for App.
func labelsForApp(name string) map[string]string {
    return map[string]string{"app_name": "app", "app_cr": name}
}


func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Complete(r)
}

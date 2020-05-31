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
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	projectv1alpha1 "github.com/fydrah/kube-projects/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=project.fydrah.com,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=project.fydrah.com,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("project", req.Name)

	var p projectv1alpha1.Project

	if err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &p); err != nil {
		return ctrl.Result{}, nil
	}

	fName := "project.finalizers.fydrah.com"

	if p.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(p.ObjectMeta.Finalizers, fName) {
			p.ObjectMeta.Finalizers = append(p.ObjectMeta.Finalizers, fName)
			if err := r.Update(ctx, &p); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(p.ObjectMeta.Finalizers, fName) {
			if err := r.DeleteExternalResources(ctx, &p); err != nil {
				return ctrl.Result{}, err
			}

			p.ObjectMeta.Finalizers = removeString(p.ObjectMeta.Finalizers, fName)
			if err := r.Update(ctx, &p); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop now since object is being deleted
		return ctrl.Result{}, nil
	}

	var ns v1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &ns); err != nil {
		log.Info("namespace not found, creating now")
		ns = v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Name,
			},
		}
		r.Create(ctx, &ns)
	}
	nsRef, err := ref.GetReference(r.Scheme, &ns)
	if err != nil {
		log.Error(err, "unable to fetch Namespace reference")
		return ctrl.Result{}, err
	}
	p.Status.Namespace = *nsRef

	var lr v1.LimitRange
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Name}, &lr); err != nil {
		log.Info("LimitRange not found, creating now")
		lr := v1.LimitRange{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Name,
			},
			Spec: p.Spec.LimitRange,
		}
		if e := r.Create(ctx, &lr); e != nil {
			log.Error(e, "unable to create LimitRange")
			return ctrl.Result{}, e
		}
	}
	lrRef, err := ref.GetReference(r.Scheme, &lr)
	if err != nil {
		log.Error(err, "unable to fetch LimitRange reference")
		return ctrl.Result{}, err
	}
	p.Status.LimitRange = *lrRef

	var rq v1.ResourceQuota
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Name}, &rq); err != nil {
		log.Info("ResourceQuota not found, creating now")
		rq := v1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Name,
			},
			Spec: p.Spec.ResourceQuota,
		}
		if e := r.Create(ctx, &rq); e != nil {
			log.Error(e, "unable to create ResourceQuota")
			return ctrl.Result{}, e
		}
	}
	rqRef, err := ref.GetReference(r.Scheme, &rq)
	if err != nil {
		log.Error(err, "unable to fetch ResourceQuota reference")
		return ctrl.Result{}, err
	}
	p.Status.LimitRange = *rqRef

	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) DeleteExternalResources(ctx context.Context, p *projectv1alpha1.Project) error {
	var ns v1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: p.ObjectMeta.Name}, &ns); err != nil {
		return fmt.Errorf("unable to get object: %v", err)
	}
	if err := r.Delete(ctx, &ns); err != nil {
		return fmt.Errorf("unable to delete object: %v", err)
	}
	return nil
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&projectv1alpha1.Project{}).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

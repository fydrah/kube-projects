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
	"reflect"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
				log.Error(err, "unable to update finalizers")
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(p.ObjectMeta.Finalizers, fName) {
			log.Info("deleting project")

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

	if err := r.CreateOrUpdateExternalResources(ctx, &p); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) CreateOrUpdateExternalResources(ctx context.Context, p *projectv1alpha1.Project) error {
	log := r.Log.WithValues("project", p.Name)
	log.Info(fmt.Sprintf("processing project %v", p.Name))
	// Namespace
	nsB := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Name,
		},
	}
	var nsF v1.Namespace
	// Namespace: Create and store ref
	if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, &nsF); err != nil {
		if errors.IsNotFound(err) {
			log.Info("creating Namespace")
			if e := r.Create(ctx, &nsB); e != nil {
				log.Error(e, "unable to create Namespace")
				return e
			}
			nsRef, e := ref.GetReference(r.Scheme, &nsB)
			if e != nil {
				log.Error(e, "unable to fetch Namespace reference")
				return e
			}
			p.Status.Namespace = *nsRef
		} else {
			log.Error(err, "unable to create Namespace")
			return err
		}
	}

	// LimitRange
	lrB := v1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Name,
		},
		Spec: v1.LimitRangeSpec(p.Spec.LimitRange),
	}
	var (
		lrF         v1.LimitRange
		lrSpecEmpty v1.LimitRangeSpec
		lrCreated   bool
	)
	// LimitRange: Create and store ref
	if err := r.Get(ctx, client.ObjectKey{Name: p.Name, Namespace: p.Name}, &lrF); err != nil {
		if errors.IsNotFound(err) {
			if !reflect.DeepEqual(lrB.Spec, lrSpecEmpty) {
				log.Info("creating LimitRange")
				if e := r.Create(ctx, &lrB); e != nil {
					log.Error(e, "unable to create LimitRange")
					return e
				}
				lrRef, e := ref.GetReference(r.Scheme, &lrB)
				if e != nil {
					log.Error(e, "unable to fetch LimitRange reference")
					return e
				}
				p.Status.LimitRange = *lrRef
				lrCreated = true
			}
		} else {
			log.Error(err, "unable to fetch LimitRange")
			return err
		}
	}
	// LimitRange: Update or Delete
	if !reflect.DeepEqual(lrB.Spec, lrF.Spec) && !lrCreated {
		if reflect.DeepEqual(lrSpecEmpty, lrB.Spec) {
			log.Info("deleting LimitRange")
			if err := r.Delete(ctx, &lrF); err != nil {
				log.Error(err, "unable to delete LimitRange")
				return err
			}
			p.Status.LimitRange = v1.ObjectReference{}
		} else {
			log.Info("updating LimitRange")
			if err := r.Update(ctx, &lrB); err != nil {
				log.Error(err, "unable to update LimitRange")
				return err
			}
		}
	}

	// Resource Quota
	rqB := v1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Name,
		},
		Spec: p.Spec.ResourceQuota,
	}
	var (
		rqF         v1.ResourceQuota
		rqSpecEmpty v1.ResourceQuotaSpec
		rqCreated   bool
	)
	// Resource Quota: Create and store ref
	if err := r.Get(ctx, client.ObjectKey{Name: p.Name, Namespace: p.Name}, &rqF); err != nil {
		if errors.IsNotFound(err) {
			if !reflect.DeepEqual(rqB.Spec, rqSpecEmpty) {
				log.Info("creating ResourceQuota")
				if e := r.Create(ctx, &rqB); e != nil {
					log.Error(e, "unable to create ResourceQuota")
					return e
				}
				rqRef, e := ref.GetReference(r.Scheme, &rqB)
				if e != nil {
					log.Error(e, "unable to fetch ResourceQuota reference")
					return e
				}
				p.Status.ResourceQuota = *rqRef
				rqCreated = true
			}
		} else {
			log.Error(err, "unable to create ResourceQuota")
			return err
		}
	}
	// Resource Quota: Update or Delete
	if !reflect.DeepEqual(rqB.Spec, rqF.Spec) && !rqCreated {
		if reflect.DeepEqual(rqSpecEmpty, rqB.Spec) {
			log.Info("deleting ResourceQuota")
			if err := r.Delete(ctx, &rqF); err != nil {
				log.Error(err, "unable to delete ResourceQuota")
				return err
			}
			p.Status.ResourceQuota = v1.ObjectReference{}
		} else {
			log.Info("updating ResourceQuota")
			if err := r.Update(ctx, &rqB); err != nil {
				log.Error(err, "unable to update ResourceQuota")
				return err
			}
		}
	}

	if err := r.Status().Update(ctx, p); err != nil {
		log.Error(err, "unable to update Project status")
		return err
	}

	return nil
}

func (r *ProjectReconciler) DeleteExternalResources(ctx context.Context, p *projectv1alpha1.Project) error {
	log := r.Log.WithValues("project", p.Name)
	lr := v1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Name,
		},
		Spec: v1.LimitRangeSpec(p.Spec.LimitRange),
	}
	if err := r.Get(ctx, client.ObjectKey{Name: p.Name, Namespace: p.Name}, &lr); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		log.Info("unable to find LimitRange")
	} else {
		if e := r.Delete(ctx, &lr); e != nil {
			return fmt.Errorf("unable to delete LimitRange: %v", e)
		}
		log.Info("limitRange deleted")
		p.Status.LimitRange = v1.ObjectReference{}
	}

	rq := v1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Name,
		},
		Spec: p.Spec.ResourceQuota,
	}
	if err := r.Get(ctx, client.ObjectKey{Name: p.Name, Namespace: p.Name}, &rq); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		log.Info("unable to find ResourceQuota")
	} else {
		if e := r.Delete(ctx, &rq); e != nil {
			return fmt.Errorf("unable to delete ResourceQuota: %v", e)
		}
		log.Info("resourceQuota deleted")
		p.Status.ResourceQuota = v1.ObjectReference{}
	}

	ns := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Name,
		},
	}
	if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, &ns); err != nil {
		return fmt.Errorf("unable to get object: %v", err)
	} else {
		if e := r.Delete(ctx, &ns); e != nil {
			return fmt.Errorf("unable to delete Namespace: %v", e)
		}
		log.Info("namespace deleted")
		p.Status.Namespace = v1.ObjectReference{}
	}

	if err := r.Status().Update(ctx, p); err != nil {
		log.Error(err, "unable to update Project status")
		return err
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

package appa

import (
	"fmt"

	appv1alpha1 "github.com/michaelhenkel/app-operator/pkg/apis/app/v1alpha1"
	utils "github.com/michaelhenkel/app-operator/pkg/controller/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_appa")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new AppA Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAppA{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("appa-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	err = c.Watch(
		&source.Kind{Type: &appv1alpha1.AppA{}},
		utils.AppHandler(utils.AppAGroupKind()),
	)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &appv1alpha1.AppB{}},
		utils.AppHandler(utils.AppBGroupKind()),
		utils.AppSizeChange(utils.AppBGroupKind()),
	)
	if err != nil {
		// handle it
	}

	return nil
}

// blank assignment to verify that ReconcileAppA implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileAppA{}

// ReconcileAppA reconciles a AppA object
type ReconcileAppA struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reconciles
func (r *ReconcileAppA) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling AppA")

	object, groupKind, err := utils.GetObjectAndGroupKindFromRequest(&request, r.client)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	switch *groupKind {
	case utils.AppAGroupKind():
		instance := object.(*appv1alpha1.AppA)
		fmt.Println("Instance: ", instance.Name)
	case utils.AppBGroupKind():
		instance := object.(*appv1alpha1.AppB)
		fmt.Println("Instance: ", instance.Name)
	}

	return reconcile.Result{}, nil
}

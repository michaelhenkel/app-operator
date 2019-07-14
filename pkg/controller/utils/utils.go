package utils

import (
	"context"
	"strings"

	appv1alpha1 "github.com/michaelhenkel/app-operator/pkg/apis/app/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// const defines the condsts
const (
	APPA = "AppA.app.example.com"
	APPB = "AppB.app.example.com"
)

// GetGroupKindFromObject return GK
func GetGroupKindFromObject(object runtime.Object) schema.GroupKind {
	objectKind := object.GetObjectKind()
	objectGroupVersionKind := objectKind.GroupVersionKind()
	return objectGroupVersionKind.GroupKind()
}

// AppHandler handles
func AppHandler(appGroupKind schema.GroupKind) handler.Funcs {
	appHandler := handler.Funcs{
		CreateFunc: func(e event.CreateEvent, q workqueue.RateLimitingInterface) {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      appGroupKind.String() + "/" + e.Meta.GetName(),
				Namespace: e.Meta.GetNamespace(),
			}})
		},
		UpdateFunc: func(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      appGroupKind.String() + "/" + e.MetaNew.GetName(),
				Namespace: e.MetaNew.GetNamespace(),
			}})
		},
		DeleteFunc: func(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      appGroupKind.String() + "/" + e.Meta.GetName(),
				Namespace: e.Meta.GetNamespace(),
			}})
		},
		GenericFunc: func(e event.GenericEvent, q workqueue.RateLimitingInterface) {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      appGroupKind.String() + "/" + e.Meta.GetName(),
				Namespace: e.Meta.GetNamespace(),
			}})
		},
	}
	return appHandler
}

// AppAGroupKind returns group kind
func AppAGroupKind() schema.GroupKind {
	return schema.ParseGroupKind(APPA)
}

// AppBGroupKind returns group kind
func AppBGroupKind() schema.GroupKind {
	return schema.ParseGroupKind(APPB)
}

// AppSizeChange returns
func AppSizeChange(appGroupKind schema.GroupKind) predicate.Funcs {
	pred := predicate.Funcs{}
	switch appGroupKind {
	case AppAGroupKind():
		pred = predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldInstance := e.ObjectOld.(*appv1alpha1.AppA)
				newInstance := e.ObjectNew.(*appv1alpha1.AppA)
				return oldInstance.Spec.Size != newInstance.Spec.Size
			},
		}
	case AppBGroupKind():
		pred = predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldInstance := e.ObjectOld.(*appv1alpha1.AppB)
				newInstance := e.ObjectNew.(*appv1alpha1.AppB)
				return oldInstance.Spec.Size != newInstance.Spec.Size
			},
		}

	}
	return pred
}

//GetObjectAndGroupKindFromRequest returns Object and Kind
func GetObjectAndGroupKindFromRequest(request *reconcile.Request, client client.Client) (runtime.Object, *schema.GroupKind, error) {
	appGroupKind := schema.ParseGroupKind(strings.Split(request.Name, "/")[0])
	appName := strings.Split(request.Name, "/")[1]
	var instance runtime.Object
	switch appGroupKind {
	case AppAGroupKind():
		instance = &appv1alpha1.AppA{}
	case AppBGroupKind():
		instance = &appv1alpha1.AppB{}
	}
	request.Name = appName
	err := client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil, err
		}
		return nil, nil, err
	}
	return instance, &appGroupKind, nil
}

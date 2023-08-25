package utils

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/meta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CreateOrUpdate creates or updates the object
func CreateOrUpdate(ctx context.Context, c client.Client, obj client.Object) (controllerutil.OperationResult, error) {
	// a copy of new object
	modifiedObj := obj.DeepCopyObject().(client.Object)
	log.Debug().Msgf("Modified: %v", modifiedObj)

	key := client.ObjectKeyFromObject(obj)
	gvk := obj.GetObjectKind().GroupVersionKind()
	if err := c.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error().Msgf("Get Object %v, %s err: %s", gvk, key, err)
			return controllerutil.OperationResultNone, err
		}
		log.Debug().Msgf("Creating Object %v, %s ...", gvk, key)
		if err := c.Create(ctx, obj); err != nil {
			log.Error().Msgf("Create Object %s err: %s", key, err)
			return controllerutil.OperationResultNone, err
		}

		log.Debug().Msgf("Object %v, %s is created successfully.", gvk, key)
		return controllerutil.OperationResultCreated, nil
	}
	log.Debug().Msgf("Found Object %v, %s: %v", gvk, key, obj)

	result := controllerutil.OperationResultNone
	if !reflect.DeepEqual(obj, modifiedObj) {
		log.Debug().Msgf("Patching Object %v, %s ...", gvk, key)
		patchData, err := client.Merge.Data(modifiedObj)
		if err != nil {
			log.Error().Msgf("Create ApplyPatch err: %s", err)
			return controllerutil.OperationResultNone, err
		}
		log.Debug().Msgf("Patch data = \n\n%s\n\n", string(patchData))

		// Only issue a Patch if the before and after resources differ
		if err := c.Patch(
			ctx,
			obj,
			client.RawPatch(types.MergePatchType, patchData),
			&client.PatchOptions{FieldManager: "fsm"},
		); err != nil {
			log.Error().Msgf("Patch Object %v, %s err: %s", gvk, key, err)
			return result, err
		}
		result = controllerutil.OperationResultUpdated
	}

	log.Debug().Msgf("Object %v, %s is %s successfully.", gvk, key, result)
	return result, nil
}

func CreateOrUpdateUnstructured(ctx context.Context, dynamicClient dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) error {
	// a copy of new object
	modifiedObj := obj.DeepCopyObject().(*unstructured.Unstructured)
	log.Debug().Msgf("Modified: %v", modifiedObj)

	key := client.ObjectKeyFromObject(obj)
	gvk := obj.GetObjectKind().GroupVersionKind()

	oldObj, err := getUnstructured(ctx, dynamicClient, mapper, obj)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error().Msgf("Get Object %v, %s err: %s", gvk, key, err)
			return err
		}

		log.Debug().Msgf("Creating Object %v, %s/%s ...", gvk, obj.GetNamespace(), obj.GetName())
		if _, err := createUnstructured(ctx, dynamicClient, mapper, obj); err != nil {
			log.Error().Msgf("Create Object %s err: %s", key, err)
			return err
		}

		log.Debug().Msgf("Object %v, %s is created successfully.", gvk, key)

		return nil
	}
	log.Debug().Msgf("Found Object %v, %s: %v", gvk, key, oldObj)

	if !reflect.DeepEqual(oldObj, modifiedObj) {
		log.Debug().Msgf("Patching Object %v, %s/%s ...", gvk, obj.GetNamespace(), obj.GetName())
		if _, err := patchUnstructured(ctx, dynamicClient, mapper, obj, modifiedObj); err != nil {
			log.Error().Msgf("Patch Object %v, %s err: %s", gvk, key, err)
			return err
		}
	}

	return nil
}

func getUnstructured(ctx context.Context, dynamicClient dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	dri, err := getDynamicResourceInterface(obj, mapper, dynamicClient)
	if err != nil {
		return nil, err
	}

	return dri.Get(ctx, obj.GetName(), metav1.GetOptions{})
}

func createUnstructured(ctx context.Context, dynamicClient dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	dri, err := getDynamicResourceInterface(obj, mapper, dynamicClient)
	if err != nil {
		return nil, err
	}

	return dri.Create(ctx, obj, metav1.CreateOptions{})
}

func patchUnstructured(ctx context.Context, dynamicClient dynamic.Interface, mapper meta.RESTMapper, obj, modifiedObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	dri, err := getDynamicResourceInterface(obj, mapper, dynamicClient)
	if err != nil {
		return nil, err
	}

	patchData, err := client.Merge.Data(modifiedObj)
	if err != nil {
		log.Error().Msgf("Create ApplyPatch err: %s", err)
		return nil, err
	}
	log.Debug().Msgf("Patch data = %s", string(patchData))

	// Only issue a Patch if the before and after resources differ
	return dri.Patch(ctx, obj.GetName(), types.MergePatchType, patchData, metav1.PatchOptions{FieldManager: "fsm"})
}

func getDynamicResourceInterface(obj *unstructured.Unstructured, mapper meta.RESTMapper, dynamicClient dynamic.Interface) (dynamic.ResourceInterface, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(corev1.NamespaceDefault)
		}
		return dynamicClient.Resource(mapping.Resource).Namespace(obj.GetNamespace()), nil
	}

	return dynamicClient.Resource(mapping.Resource), nil
}

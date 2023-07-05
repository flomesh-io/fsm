package utils

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateOrUpdate(ctx context.Context, c client.Client, obj client.Object) (controllerutil.OperationResult, error) {
	// a copy of new object
	modifiedObj := obj.DeepCopyObject().(client.Object)
	klog.V(5).Infof("Modified: %v", modifiedObj)

	key := client.ObjectKeyFromObject(obj)
	gvk := obj.GetObjectKind().GroupVersionKind()
	if err := c.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("Get Object %v, %s err: %s", gvk, key, err)
			return controllerutil.OperationResultNone, err
		}
		klog.V(5).Infof("Creating Object %v, %s ...", gvk, key)
		if err := c.Create(ctx, obj); err != nil {
			klog.Errorf("Create Object %s err: %s", key, err)
			return controllerutil.OperationResultNone, err
		}

		klog.V(5).Infof("Object %v, %s is created successfully.", gvk, key)
		return controllerutil.OperationResultCreated, nil
	}
	klog.V(5).Infof("Found Object %v, %s: %v", gvk, key, obj)

	result := controllerutil.OperationResultNone
	if !reflect.DeepEqual(obj, modifiedObj) {
		klog.V(5).Infof("Patching Object %v, %s ...", gvk, key)
		patchData, err := client.Merge.Data(modifiedObj)
		if err != nil {
			klog.Errorf("Create ApplyPatch err: %s", err)
			return controllerutil.OperationResultNone, err
		}
		klog.V(5).Infof("Patch data = \n\n%s\n\n", string(patchData))

		// Only issue a Patch if the before and after resources differ
		if err := c.Patch(
			ctx,
			obj,
			client.RawPatch(types.MergePatchType, patchData),
			&client.PatchOptions{FieldManager: "fsm"},
		); err != nil {
			klog.Errorf("Patch Object %v, %s err: %s", gvk, key, err)
			return result, err
		}
		result = controllerutil.OperationResultUpdated
	}

	klog.V(5).Infof("Object %v, %s is %s successfully.", gvk, key, result)
	return result, nil
}

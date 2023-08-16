package utils

import (
	"context"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CreateOrUpdate creates or updates the object
func CreateOrUpdate(ctx context.Context, c client.Client, obj client.Object) (controllerutil.OperationResult, error) {
	// a copy of new object
	modifiedObj := obj.DeepCopyObject().(client.Object)
	log.Info().Msgf("Modified: %v", modifiedObj)

	key := client.ObjectKeyFromObject(obj)
	gvk := obj.GetObjectKind().GroupVersionKind()
	if err := c.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error().Msgf("Get Object %v, %s err: %s", gvk, key, err)
			return controllerutil.OperationResultNone, err
		}
		log.Info().Msgf("Creating Object %v, %s ...", gvk, key)
		if err := c.Create(ctx, obj); err != nil {
			log.Error().Msgf("Create Object %s err: %s", key, err)
			return controllerutil.OperationResultNone, err
		}

		log.Info().Msgf("Object %v, %s is created successfully.", gvk, key)
		return controllerutil.OperationResultCreated, nil
	}
	log.Info().Msgf("Found Object %v, %s: %v", gvk, key, obj)

	result := controllerutil.OperationResultNone
	if !reflect.DeepEqual(obj, modifiedObj) {
		log.Info().Msgf("Patching Object %v, %s ...", gvk, key)
		patchData, err := client.Merge.Data(modifiedObj)
		if err != nil {
			log.Error().Msgf("Create ApplyPatch err: %s", err)
			return controllerutil.OperationResultNone, err
		}
		log.Info().Msgf("Patch data = \n\n%s\n\n", string(patchData))

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

	log.Info().Msgf("Object %v, %s is %s successfully.", gvk, key, result)
	return result, nil
}

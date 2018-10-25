package configutil

import (
	"reflect"
	"unsafe"
)

type pointerInterface struct {
	Type, Data unsafe.Pointer
}

// Merge deeply merges two objects
// object MUST be a pointer of a pointer
// overwriteObject MUST be a pointer
func Merge(object interface{}, overwriteObject interface{}) {
	objectPointerUnsafe := (*(*pointerInterface)(unsafe.Pointer(&object))).Data
	overwriteObjectPointerUnsafe := (*(*pointerInterface)(unsafe.Pointer(&overwriteObject))).Data

	merge(object, overwriteObject, objectPointerUnsafe, overwriteObjectPointerUnsafe)
}

func merge(objectPointer interface{}, overwriteObjectPointer interface{}, objectPointerUnsafe unsafe.Pointer, overwriteObjectPointerUnsafe unsafe.Pointer) {
	overwriteObjectRef := reflect.ValueOf(overwriteObjectPointer)

	if !overwriteObjectRef.IsNil() {
		if overwriteObjectRef.Kind() == reflect.Ptr {
			overwriteObjectRef = overwriteObjectRef.Elem()
		}
		objectPointerReal := reflect.ValueOf(objectPointer).Elem().Interface()
		overwriteObject := overwriteObjectRef.Interface()
		overwriteObjectType := reflect.TypeOf(overwriteObject)
		overwriteObjectKind := overwriteObjectType.Kind()
		objectPointerRef := reflect.ValueOf(objectPointerReal)
		var objectRef reflect.Value

		if !objectPointerRef.IsNil() {
			objectRef = reflect.ValueOf(objectPointerReal).Elem()
		}

		switch overwriteObjectKind {
		case reflect.Slice:
			if objectPointerRef.IsNil() {
				objectRef.Set(reflect.New(overwriteObjectType))
			}

			for i := 0; i < overwriteObjectRef.Len(); i++ {
				overwriteValue := overwriteObjectRef.Index(i)

				objectRef.Set(reflect.Append(objectRef, overwriteValue))
			}
		case reflect.Map:
			if objectPointerRef.IsNil() {
				objectRef.Set(overwriteObjectRef)
			} else {
				genericPointerType := reflect.TypeOf(overwriteObject)

				for _, keyRef := range overwriteObjectRef.MapKeys() {
					key := keyRef.Interface()
					overwriteValue := getMapValue(overwriteObject, key, genericPointerType)
					valuePointerRef := objectRef.MapIndex(keyRef)

					if isZero(valuePointerRef) == false {
						valuePointer := valuePointerRef.Interface()

						Merge(&valuePointer, overwriteValue)
					} else {
						keyRef := reflect.ValueOf(key)
						overwriteValueRef := reflect.ValueOf(overwriteValue)

						objectRef.SetMapIndex(keyRef, overwriteValueRef)
					}
				}
			}
		case reflect.Struct:
			for i := 0; i < overwriteObjectRef.NumField(); i++ {
				overwriteValueRef := overwriteObjectRef.Field(i)
				overwriteValuePointerRef := reflect.ValueOf(overwriteValueRef.Interface())

				if !overwriteValuePointerRef.IsNil() {
					overwriteValue := overwriteValueRef.Interface()
					valuePointerRef := objectRef.Field(i)

					if valuePointerRef.IsNil() {
						valuePointerRef.Set(reflect.ValueOf(overwriteValue))
					} else {
						valuePointer := valuePointerRef.Interface()

						Merge(&valuePointer, overwriteValue)
					}
				}
			}
		default:
			*(*unsafe.Pointer)(objectPointerUnsafe) = overwriteObjectPointerUnsafe
		}
	}
}

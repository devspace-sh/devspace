package configutil

import (
	"reflect"
	"unsafe"

	"gopkg.in/yaml.v2"
)

type pointerInterface struct {
	Type, Data unsafe.Pointer
}

// Merge deeply merges two objects
// object MUST be a pointer of a pointer
// overwriteObject MUST be a pointer
func Merge(object interface{}, overwriteObject interface{}, unifyPointers bool) {
	overwriteObjectRef := reflect.ValueOf(overwriteObject)

	if !overwriteObjectRef.IsNil() {
		if overwriteObjectRef.Kind() == reflect.Ptr {
			overwriteObjectRef = overwriteObjectRef.Elem()
		}
		objectPointerReal := reflect.ValueOf(object).Elem().Interface()
		overwriteObject := overwriteObjectRef.Interface()

		if !unifyPointers {
			overwriteObject = deepCopy(overwriteObject)

			// ensure deepCopy only runs once
			unifyPointers = true
		}
		overwriteObjectType := reflect.TypeOf(overwriteObject)
		overwriteObjectKind := overwriteObjectType.Kind()
		objectPointerRef := reflect.ValueOf(objectPointerReal)

		var objectRef reflect.Value

		if !objectPointerRef.IsNil() {
			objectRef = objectPointerRef.Elem()
		}

		switch overwriteObjectKind {
		case reflect.Slice:
			objectRef.Set(overwriteObjectRef)
		case reflect.Map:
			if objectPointerRef.IsNil() {
				objectRef.Set(overwriteObjectRef)
			} else {
				genericPointerType := reflect.TypeOf(overwriteObject)

				for _, keyRef := range overwriteObjectRef.MapKeys() {
					key := keyRef.Interface()
					overwriteValue := getMapValue(overwriteObject, key, genericPointerType)
					valuePointerRef := objectRef.MapIndex(keyRef)

					if isZero(valuePointerRef) == false && !isTrivialDataType(valuePointerRef) {
						valuePointer := valuePointerRef.Interface()

						Merge(&valuePointer, overwriteValue, unifyPointers)
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

				if !overwriteValueRef.IsNil() {
					overwriteValue := overwriteValueRef.Interface()
					valuePointerRef := objectRef.Field(i)

					if valuePointerRef.IsNil() || isTrivialDataType(valuePointerRef) {
						valuePointerRef.Set(overwriteValueRef)
					} else {
						valuePointer := valuePointerRef.Interface()

						Merge(&valuePointer, overwriteValue, unifyPointers)
					}
				}
			}
		default:
			objectPointerUnsafe := (*(*pointerInterface)(unsafe.Pointer(&object))).Data
			overwriteObjectPointerUnsafe := (*(*pointerInterface)(unsafe.Pointer(&overwriteObject))).Data

			*(*unsafe.Pointer)(objectPointerUnsafe) = overwriteObjectPointerUnsafe
		}
	}
}

func isTrivialDataType(value reflect.Value) bool {
	valueType := value.Type()

	if valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}

	switch valueType.Kind() {
	case reflect.Slice:
		return false
	case reflect.Map:
		return false
	case reflect.Struct:
		return false
	}
	return true
}

func deepCopy(from interface{}) interface{} {
	yamlData, _ := yaml.Marshal(from)
	objectType := reflect.TypeOf(from)

	copy := reflect.New(objectType).Interface()

	yaml.Unmarshal(yamlData, copy)

	return copy
}

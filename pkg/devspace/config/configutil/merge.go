package configutil

import (
	"reflect"
	"unsafe"
)

func merge(objectPointer interface{}, overwriteObjectPointer interface{}, objectPointerUnsafe unsafe.Pointer, overwriteObjectPointerUnsafe unsafe.Pointer) {
	overwriteObjectRef := reflect.ValueOf(overwriteObjectPointer)

	if !overwriteObjectRef.IsNil() {
		if overwriteObjectRef.Kind() == reflect.Ptr {
			overwriteObjectRef = overwriteObjectRef.Elem()
		}
		overwriteObject := overwriteObjectRef.Interface()
		overwriteObjectType := reflect.TypeOf(overwriteObject)
		overwriteObjectKind := overwriteObjectType.Kind()
		objectPointerRef := reflect.ValueOf(objectPointer)
		var objectRef reflect.Value

		if !objectPointerRef.IsNil() {
			objectRef = reflect.ValueOf(objectPointer).Elem()
		}

		switch overwriteObjectKind {
		case reflect.Slice:
			if objectPointerRef.IsNil() {
				objectRef.Set(overwriteObjectRef)
			} else {
				for i := 0; i < overwriteObjectRef.Len(); i++ {
					overwriteValue := overwriteObjectRef.Index(i)

					objectRef.Set(reflect.Append(objectRef, overwriteValue))
				}
			}
		case reflect.Map:
			var mergedMap map[interface{}]interface{}

			if !objectPointerRef.IsNil() {
				objectRef.Set(overwriteObjectRef)
			} else {
				mergedMap = map[interface{}]interface{}{}

				overwriteMap := overwriteObject.(map[interface{}]interface{})

				for key, overwriteValue := range overwriteMap {
					valuePointer, keyExists := mergedMap[key]

					valuePointerRef := reflect.ValueOf(valuePointer)

					if keyExists && !valuePointerRef.IsNil() {
						merge(valuePointer, overwriteValue, unsafe.Pointer(&valuePointer), unsafe.Pointer(&overwriteValue))
					} else {
						keyRef := reflect.ValueOf(key)
						overwriteValueRef := reflect.ValueOf(overwriteValue)

						objectRef.SetMapIndex(keyRef, overwriteValueRef)
					}
				}
			}
		case reflect.Struct:
			for i := 0; i < overwriteObjectRef.NumField(); i++ {
				//fieldName := objectRef.Type().Field(i).Name
				overwriteValueRef := overwriteObjectRef.Field(i)
				overwriteValuePointerRef := reflect.ValueOf(overwriteValueRef.Interface())

				if !overwriteValuePointerRef.IsNil() {
					overwriteValue := overwriteValueRef.Interface()
					valuePointerRef := objectRef.Field(i)

					if valuePointerRef.IsNil() {
						objectRef.Field(i).Set(reflect.ValueOf(overwriteValue))
					} else {
						valuePointer := objectRef.Field(i).Interface()

						merge(valuePointer, overwriteValue, unsafe.Pointer(&valuePointer), unsafe.Pointer(&overwriteValue))
					}
				}
			}
		default:
			*(*unsafe.Pointer)(objectPointerUnsafe) = overwriteObjectPointerUnsafe
		}
	}
}

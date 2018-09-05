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

		switch overwriteObjectKind {
		case reflect.Slice:
		case reflect.Struct:
			objectValues := reflect.ValueOf(objectPointer).Elem()
			overwriteObjectValues := reflect.ValueOf(overwriteObjectPointer).Elem()

			for i := 0; i < overwriteObjectValues.NumField(); i++ {
				//fieldName := objectValues.Type().Field(i).Name
				overwriteValueRef := overwriteObjectValues.Field(i)
				overwriteValuePointerRef := reflect.ValueOf(overwriteValueRef.Interface())

				if !overwriteValuePointerRef.IsNil() {
					overwriteValue := overwriteValueRef.Interface()
					valuePointerRef := objectValues.Field(i)

					if valuePointerRef.IsNil() {
						objectValues.Field(i).Set(reflect.ValueOf(overwriteValue))
					} else {
						valuePointer := objectValues.Field(i).Interface()

						merge(valuePointer, overwriteValue, unsafe.Pointer(&valuePointer), unsafe.Pointer(&overwriteValue))
					}
				}
			}
		default:
			*(*unsafe.Pointer)(objectPointerUnsafe) = overwriteObjectPointerUnsafe
		}
	}
}

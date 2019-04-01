package configutil

import (
	"reflect"
	"strings"
	"unsafe"

	yaml "gopkg.in/yaml.v2"
)

type pointerInterface struct {
	Type, Data unsafe.Pointer
}

// Merge deeply merges two objects
// object MUST be a pointer of a pointer
// overwriteObject MUST be a pointer
func Merge(object interface{}, overwriteObject interface{}) {
	overwriteObjectRef := reflect.ValueOf(overwriteObject)

	if !overwriteObjectRef.IsNil() {
		if overwriteObjectRef.Kind() == reflect.Ptr {
			overwriteObjectRef = overwriteObjectRef.Elem()
		}
		objectPointerReal := reflect.ValueOf(object).Elem().Interface()

		overwriteObject := overwriteObjectRef.Interface()
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

				if !overwriteValueRef.IsNil() {
					overwriteValue := overwriteValueRef.Interface()
					valuePointerRef := objectRef.Field(i)

					if valuePointerRef.IsNil() || isTrivialDataType(valuePointerRef) {
						valuePointerRef.Set(overwriteValueRef)
					} else {
						valuePointer := valuePointerRef.Interface()

						Merge(&valuePointer, overwriteValue)
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

func getMapValue(valueMap interface{}, key interface{}, refType reflect.Type) interface{} {
	valueMapValue, _ := getPointerValue(valueMap)
	mapRef := reflect.ValueOf(valueMapValue)
	keyRef := reflect.ValueOf(key)
	mapValue := mapRef.MapIndex(keyRef)

	if isZero(mapValue) {
		mapValue = reflect.New(refType)
	}
	return mapValue.Interface()
}

//isZero is a reflect function from: https://github.com/golang/go/issues/7501
func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func getYamlKey(key string) string {
	re := ""

	for i := 0; i < len(key); i++ {
		letter := key[i : i+1]
		lowerLetter := strings.ToLower(letter)

		if i == 0 || (letter != lowerLetter) {
			re = re + lowerLetter
		} else {
			re = re + key[i:]
			break
		}
	}
	return re
}

func getPointerValue(object interface{}) (interface{}, bool) {
	if object != nil {
		objectType := reflect.TypeOf(object)
		objectKind := objectType.Kind()

		if objectKind == reflect.Ptr {
			objectValueRef := reflect.ValueOf(object)

			if objectValueRef.IsNil() {
				pointerValueType := objectValueRef.Type().Elem()
				newInstance, _ := getPointerValue(reflect.New(pointerValueType).Interface())

				return newInstance, true
			}
			value, _ := getPointerValue(objectValueRef.Elem().Interface())

			return value, false
		}
	}
	return object, false
}
